import { BookOpen } from "lucide-react";
import helpHtml from "../../../help.md?html";

export function HelpPanel({ onClose }: { onClose: () => void }) {
  return (
    <div className="modal-backdrop sponsor-backdrop" onClick={(event) => { if (event.target === event.currentTarget) onClose(); }}>
      <section className="help-modal" onClick={(event) => event.stopPropagation()}>
        <div className="modal-title sponsor-title">
          <div>
            <h2><BookOpen size={20} /> 游戏说明</h2>
            <p className="hint">了解六款游戏规则、全局玩法机制与页面功能。</p>
          </div>
          <button type="button" className="icon-button" onClick={onClose}>×</button>
        </div>
        <div className="help-content" dangerouslySetInnerHTML={{ __html: helpHtml }} />
      </section>
    </div>
  );
}
