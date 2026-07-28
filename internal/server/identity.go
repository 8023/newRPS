package server

import "strings"

// 多端身份认领：认领密钥（ClaimKey）与长期设备凭据（PlayerSecrets）是两个不同的东西。
//   - PlayerSecrets：每台已登录设备一条，明文，永久有效（除非登出时撤销），
//     player:join 每次重连都要带上其中一条才能证明身份。
//   - ClaimKey：单值、明文、一次性（randomClaimKey，12 字节）——只用于把身份「搬」到一台新设备。
//     服务端/库内 playerId 始终是 UUID；前端展示认领码时把 playerId 压成 base64url，
//     粘贴恢复时再解回 UUID 再调 identity:claim。验证通过后：① 生成一条全新的
//     PlayerSecrets 供新设备使用；② 立即轮换 ClaimKey（用过即作废）。

// isSameDevice 判断新连接的 deviceKey 是否和这个玩家上次记录的 deviceKey 一致——
// 一致视为「同设备刷新/重连」，不一致视为「换了一台设备」。
func isSameDevice(recordedDeviceKey, newDeviceKey string) bool {
	return recordedDeviceKey != "" && recordedDeviceKey == newDeviceKey
}

// isLegacySecretFormat 识别前端早期版本生成的双 UUID 拼接格式（36+1+36=73 字符、
// 9 个 "-"）。命中时 onPlayerJoin 会静默换发 randomPlayerSecret（不强制用户重登）；
// 未命中的旧短 secret / 单 UUID 仍照常校验，不做迁移打断。
func isLegacySecretFormat(secret string) bool {
	return len(secret) == 73 && strings.Count(secret, "-") == 9
}

// needsKickConfirm 判断是否要先弹「已在其他设备登录」确认，而不是直接顶替。
// 同设备刷新永远不需要确认（不打断当前使用体验）；不同设备除非已经带上 forceKick
// （用户已经在前端确认过一次），否则必须先确认。
func needsKickConfirm(sameDevice, forceKick bool) bool {
	return !sameDevice && !forceKick
}

// verifyPlayerSecret 校验一段明文 secret 能否证明这就是该玩家本人：查新版明文设备列表。
func (s *Server) verifyPlayerSecret(player *PlayerState, secret string) bool {
	if secret == "" {
		return false
	}
	return player.hasPlayerSecret(secret)
}

// onIdentityShowClaimKey 把当前有效的认领密钥回给发起请求的这一条 socket（不广播）。
func (s *Server) onIdentityShowClaimKey(client *Client, env wsEnvelope) {
	player, ok := s.requirePlayer(client, env)
	if !ok {
		return
	}
	if !player.Persistent {
		client.reply(env.ID, nil, "当前身份不支持认领")
		return
	}
	if player.ClaimKey == "" {
		player.ClaimKey = randomClaimKey()
		s.requestPersist("lazy")
	}
	client.reply(env.ID, map[string]any{"claimKey": player.ClaimKey, "playerId": player.PlayerID}, "")
}

// onIdentityRefreshClaimKey 手动作废旧密钥、生成一把新的（不需要先认证旧密钥）。
// 用途：怀疑旧密钥已经泄露，或者登出前想现场生成一把给自己抄。
func (s *Server) onIdentityRefreshClaimKey(client *Client, env wsEnvelope) {
	player, ok := s.requirePlayer(client, env)
	if !ok {
		return
	}
	if !player.Persistent {
		client.reply(env.ID, nil, "当前身份不支持认领")
		return
	}
	player.ClaimKey = randomClaimKey()
	s.requestPersist("lazy")
	client.reply(env.ID, map[string]any{"claimKey": player.ClaimKey, "playerId": player.PlayerID}, "")
}

// onIdentityClaim 用另一台设备展示的 playerId+claimKey 认领身份：校验通过后生成一条全新
// 的设备凭据返回给调用方（前端拿到后覆写本地 playerId/playerSecret 并重新 player:join）。
// 不要求当前 socket 已经 join 过——这本来就是「换身份」的入口，独立于常规登录流程。
func (s *Server) onIdentityClaim(client *Client, env wsEnvelope) {
	var p struct {
		PlayerID string `json:"playerId"`
		ClaimKey string `json:"claimKey"`
	}
	_ = decodeD(env, &p)
	if p.PlayerID == "" || p.ClaimKey == "" {
		client.reply(env.ID, nil, "认领信息不完整")
		return
	}
	id := s.playerIdToID[p.PlayerID]
	player := s.players[id]
	if player == nil || !player.Persistent || player.ClaimKey == "" || player.ClaimKey != p.ClaimKey {
		s.securityLog("identity_claim_invalid", map[string]any{
			"ip": client.ipAddress, "device": client.deviceKey, "userAgent": client.userAgent,
		})
		client.reply(env.ID, nil, "认领密钥无效或已过期")
		return
	}
	newSecret := randomPlayerSecret()
	evicted := player.addPlayerSecret(newSecret)
	// 用过即作废：认领成功后立即轮换，旧密钥不能再用第二次。
	player.ClaimKey = randomClaimKey()
	s.requestPersist("lazy")
	if evicted != "" {
		s.notifyDeviceEvicted(player, evicted)
	}
	// 顺带把当前名字/性别带回去，前端接着重新 player:join 时要用这份真实值，不能用输入框里
	// 随手打的名字，否则会把认领回来的账号名字覆盖掉——阵营由性别查表决定，不用单独带回。
	client.reply(env.ID, map[string]any{
		"playerId":     player.PlayerID,
		"playerSecret": newSecret,
		"name":         player.Name,
		"genderId":     player.GenderID,
	}, "")
}

// onIdentityLogout 撤销当前设备的凭据（从 PlayerSecrets 里删掉这一条）。
// 前端在清空本地 localStorage 之前调用，让服务端也不再信任这台设备。
func (s *Server) onIdentityLogout(client *Client, env wsEnvelope) {
	var p struct {
		PlayerSecret string `json:"playerSecret"`
	}
	_ = decodeD(env, &p)
	player, ok := s.requirePlayer(client, env)
	if !ok {
		return
	}
	if p.PlayerSecret != "" {
		player.removePlayerSecret(p.PlayerSecret)
		if player.ActiveSecret == p.PlayerSecret {
			player.ActiveSecret = ""
		}
		s.requestPersist("lazy")
	}
	client.reply(env.ID, map[string]any{"ok": true}, "")
}

// notifyDeviceEvicted：设备列表满 3 条时挤掉最早一条，尽力找到那条 secret 对应的在线
// socket 并提示一下（找不到就算了——大概率那台设备本来就不在线）。
func (s *Server) notifyDeviceEvicted(player *PlayerState, evictedSecret string) {
	if player.SocketID == "" {
		return
	}
	// 当前实现无法从 secret 反查是哪个 socket（同一账号可能有多条在线连接时才需要精确定位），
	// 这里退化为：只要该账号当前有活跃连接就提示一下"有一台旧设备的登录状态已被移除"，
	// 具体是不是这条活跃连接本身不重要——反正它没被移除（活跃连接用的 secret 刚被 append，不会是被挤掉的那条）。
	s.emitToClient(player.SocketID, "identity:deviceEvicted", map[string]any{
		"message": "设备数量已达上限，最早登录的一台设备已被自动登出。",
	})
}
