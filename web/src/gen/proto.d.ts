import * as $protobuf from "protobufjs";
import Long = require("long");

/** Namespace wire. */
export namespace wire {

    /**
     * Properties of an Envelope.
     * @deprecated Use wire.Envelope.$Properties instead.
     */
    interface IEnvelope extends wire.Envelope.$Properties {
    }

    /** Represents an Envelope. */
    class Envelope {

        /**
         * Constructs a new Envelope.
         * @param [properties] Properties to set
         */
        constructor(properties?: wire.Envelope.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** Envelope event. */
        event: string;

        /** Envelope id. */
        id: (number|Long);

        /** Envelope err. */
        err: string;

        /** Envelope kind. */
        kind: wire.PayloadKind;

        /** Envelope channel. */
        channel: string;

        /** Envelope seq. */
        seq: (number|Long);

        /** Envelope hash. */
        hash: string;

        /** Envelope fullState. */
        fullState?: (game.StateDocument.$Properties|null);

        /** Envelope delta. */
        delta?: (wire.StateDelta.$Properties|null);

        /** Envelope rawBody. */
        rawBody?: (game.RawBody.$Properties|null);

        /**
         * Creates a new Envelope instance using the specified properties.
         * @param [properties] Properties to set
         * @returns Envelope instance
         */
        static create(properties: wire.Envelope.$Shape): wire.Envelope & wire.Envelope.$Shape;
        static create(properties?: wire.Envelope.$Properties): wire.Envelope;

        /**
         * Encodes the specified Envelope message. Does not implicitly {@link wire.Envelope.verify|verify} messages.
         * @param message Envelope message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: wire.Envelope.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified Envelope message, length delimited. Does not implicitly {@link wire.Envelope.verify|verify} messages.
         * @param message Envelope message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: wire.Envelope.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes an Envelope message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {wire.Envelope & wire.Envelope.$Shape} Envelope
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): wire.Envelope & wire.Envelope.$Shape;

        /**
         * Decodes an Envelope message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {wire.Envelope & wire.Envelope.$Shape} Envelope
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): wire.Envelope & wire.Envelope.$Shape;

        /**
         * Verifies an Envelope message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates an Envelope message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns Envelope
         */
        static fromObject(object: { [k: string]: any }): wire.Envelope;

        /**
         * Creates a plain object from an Envelope message. Also converts values to other types if specified.
         * @param message Envelope
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: wire.Envelope, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this Envelope to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for Envelope
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace Envelope {

        /** Properties of an Envelope. */
        interface $Properties {

            /** Envelope event */
            event?: (string|null);

            /** Envelope id */
            id?: (number|Long|null);

            /** Envelope err */
            err?: (string|null);

            /** Envelope kind */
            kind?: (wire.PayloadKind|null);

            /** Envelope channel */
            channel?: (string|null);

            /** Envelope seq */
            seq?: (number|Long|null);

            /** Envelope hash */
            hash?: (string|null);

            /** Envelope fullState */
            fullState?: (game.StateDocument.$Properties|null);

            /** Envelope delta */
            delta?: (wire.StateDelta.$Properties|null);

            /** Envelope rawBody */
            rawBody?: (game.RawBody.$Properties|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of an Envelope. */
        type $Shape = {
          event?: string|null;
          id?: number|Long|null;
          err?: string|null;
          kind?: wire.PayloadKind|null;
          channel?: string|null;
          seq?: number|Long|null;
          hash?: string|null;
          fullState?: game.StateDocument.$Shape|null;
          delta?: wire.StateDelta.$Shape|null;
          rawBody?: game.RawBody.$Shape|null;
          $unknowns?: Uint8Array[];
        };
    }

    /** PayloadKind enum. */
    enum PayloadKind {

        /** KIND_RAW value */
        KIND_RAW = 0,

        /** KIND_FULL value */
        KIND_FULL = 1,

        /** KIND_DELTA value */
        KIND_DELTA = 2
    }

    /**
     * Properties of a StateDelta.
     * @deprecated Use wire.StateDelta.$Properties instead.
     */
    interface IStateDelta extends wire.StateDelta.$Properties {
    }

    /** Represents a StateDelta. */
    class StateDelta {

        /**
         * Constructs a new StateDelta.
         * @param [properties] Properties to set
         */
        constructor(properties?: wire.StateDelta.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** StateDelta ops. */
        ops: wire.PatchOp.$Properties[];

        /**
         * Creates a new StateDelta instance using the specified properties.
         * @param [properties] Properties to set
         * @returns StateDelta instance
         */
        static create(properties: wire.StateDelta.$Shape): wire.StateDelta & wire.StateDelta.$Shape;
        static create(properties?: wire.StateDelta.$Properties): wire.StateDelta;

        /**
         * Encodes the specified StateDelta message. Does not implicitly {@link wire.StateDelta.verify|verify} messages.
         * @param message StateDelta message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: wire.StateDelta.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified StateDelta message, length delimited. Does not implicitly {@link wire.StateDelta.verify|verify} messages.
         * @param message StateDelta message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: wire.StateDelta.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a StateDelta message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {wire.StateDelta & wire.StateDelta.$Shape} StateDelta
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): wire.StateDelta & wire.StateDelta.$Shape;

        /**
         * Decodes a StateDelta message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {wire.StateDelta & wire.StateDelta.$Shape} StateDelta
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): wire.StateDelta & wire.StateDelta.$Shape;

        /**
         * Verifies a StateDelta message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a StateDelta message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns StateDelta
         */
        static fromObject(object: { [k: string]: any }): wire.StateDelta;

        /**
         * Creates a plain object from a StateDelta message. Also converts values to other types if specified.
         * @param message StateDelta
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: wire.StateDelta, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this StateDelta to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for StateDelta
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace StateDelta {

        /** Properties of a StateDelta. */
        interface $Properties {

            /** StateDelta ops */
            ops?: (wire.PatchOp.$Properties[]|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a StateDelta. */
        type $Shape = {
          ops?: wire.PatchOp.$Shape[]|null;
          $unknowns?: Uint8Array[];
        };
    }

    /**
     * Properties of a PatchOp.
     * @deprecated Use wire.PatchOp.$Properties instead.
     */
    interface IPatchOp extends wire.PatchOp.$Properties {
    }

    /** Represents a PatchOp. */
    class PatchOp {

        /**
         * Constructs a new PatchOp.
         * @param [properties] Properties to set
         */
        constructor(properties?: wire.PatchOp.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** PatchOp path. */
        path: string;

        /** PatchOp value. */
        value?: (google.protobuf.Value.$Properties|null);

        /** PatchOp remove. */
        remove: boolean;

        /**
         * Creates a new PatchOp instance using the specified properties.
         * @param [properties] Properties to set
         * @returns PatchOp instance
         */
        static create(properties: wire.PatchOp.$Shape): wire.PatchOp & wire.PatchOp.$Shape;
        static create(properties?: wire.PatchOp.$Properties): wire.PatchOp;

        /**
         * Encodes the specified PatchOp message. Does not implicitly {@link wire.PatchOp.verify|verify} messages.
         * @param message PatchOp message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: wire.PatchOp.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified PatchOp message, length delimited. Does not implicitly {@link wire.PatchOp.verify|verify} messages.
         * @param message PatchOp message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: wire.PatchOp.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a PatchOp message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {wire.PatchOp & wire.PatchOp.$Shape} PatchOp
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): wire.PatchOp & wire.PatchOp.$Shape;

        /**
         * Decodes a PatchOp message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {wire.PatchOp & wire.PatchOp.$Shape} PatchOp
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): wire.PatchOp & wire.PatchOp.$Shape;

        /**
         * Verifies a PatchOp message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a PatchOp message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns PatchOp
         */
        static fromObject(object: { [k: string]: any }): wire.PatchOp;

        /**
         * Creates a plain object from a PatchOp message. Also converts values to other types if specified.
         * @param message PatchOp
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: wire.PatchOp, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this PatchOp to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for PatchOp
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace PatchOp {

        /** Properties of a PatchOp. */
        interface $Properties {

            /** PatchOp path */
            path?: (string|null);

            /** PatchOp value */
            value?: (google.protobuf.Value.$Properties|null);

            /** PatchOp remove */
            remove?: (boolean|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a PatchOp. */
        type $Shape = {
          path?: string|null;
          value?: google.protobuf.Value.$Shape|null;
          remove?: boolean|null;
          $unknowns?: Uint8Array[];
        };
    }
}

/** Namespace game. */
export namespace game {

    /**
     * Properties of a GenderColors.
     * @deprecated Use game.GenderColors.$Properties instead.
     */
    interface IGenderColors extends game.GenderColors.$Properties {
    }

    /** Represents a GenderColors. */
    class GenderColors {

        /**
         * Constructs a new GenderColors.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.GenderColors.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** GenderColors textColor. */
        textColor: string;

        /** GenderColors backgroundColor. */
        backgroundColor: string;

        /** GenderColors borderColor. */
        borderColor: string;

        /**
         * Creates a new GenderColors instance using the specified properties.
         * @param [properties] Properties to set
         * @returns GenderColors instance
         */
        static create(properties: game.GenderColors.$Shape): game.GenderColors & game.GenderColors.$Shape;
        static create(properties?: game.GenderColors.$Properties): game.GenderColors;

        /**
         * Encodes the specified GenderColors message. Does not implicitly {@link game.GenderColors.verify|verify} messages.
         * @param message GenderColors message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.GenderColors.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified GenderColors message, length delimited. Does not implicitly {@link game.GenderColors.verify|verify} messages.
         * @param message GenderColors message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.GenderColors.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a GenderColors message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.GenderColors & game.GenderColors.$Shape} GenderColors
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.GenderColors & game.GenderColors.$Shape;

        /**
         * Decodes a GenderColors message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.GenderColors & game.GenderColors.$Shape} GenderColors
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.GenderColors & game.GenderColors.$Shape;

        /**
         * Verifies a GenderColors message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a GenderColors message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns GenderColors
         */
        static fromObject(object: { [k: string]: any }): game.GenderColors;

        /**
         * Creates a plain object from a GenderColors message. Also converts values to other types if specified.
         * @param message GenderColors
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.GenderColors, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this GenderColors to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for GenderColors
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace GenderColors {

        /** Properties of a GenderColors. */
        interface $Properties {

            /** GenderColors textColor */
            textColor?: (string|null);

            /** GenderColors backgroundColor */
            backgroundColor?: (string|null);

            /** GenderColors borderColor */
            borderColor?: (string|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a GenderColors. */
        type $Shape = game.GenderColors.$Properties;
    }

    /**
     * Properties of a GenderOption.
     * @deprecated Use game.GenderOption.$Properties instead.
     */
    interface IGenderOption extends game.GenderOption.$Properties {
    }

    /** Represents a GenderOption. */
    class GenderOption {

        /**
         * Constructs a new GenderOption.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.GenderOption.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** GenderOption id. */
        id: string;

        /** GenderOption label. */
        label: string;

        /** GenderOption factionId. */
        factionId: string;

        /**
         * Creates a new GenderOption instance using the specified properties.
         * @param [properties] Properties to set
         * @returns GenderOption instance
         */
        static create(properties: game.GenderOption.$Shape): game.GenderOption & game.GenderOption.$Shape;
        static create(properties?: game.GenderOption.$Properties): game.GenderOption;

        /**
         * Encodes the specified GenderOption message. Does not implicitly {@link game.GenderOption.verify|verify} messages.
         * @param message GenderOption message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.GenderOption.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified GenderOption message, length delimited. Does not implicitly {@link game.GenderOption.verify|verify} messages.
         * @param message GenderOption message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.GenderOption.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a GenderOption message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.GenderOption & game.GenderOption.$Shape} GenderOption
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.GenderOption & game.GenderOption.$Shape;

        /**
         * Decodes a GenderOption message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.GenderOption & game.GenderOption.$Shape} GenderOption
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.GenderOption & game.GenderOption.$Shape;

        /**
         * Verifies a GenderOption message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a GenderOption message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns GenderOption
         */
        static fromObject(object: { [k: string]: any }): game.GenderOption;

        /**
         * Creates a plain object from a GenderOption message. Also converts values to other types if specified.
         * @param message GenderOption
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.GenderOption, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this GenderOption to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for GenderOption
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace GenderOption {

        /** Properties of a GenderOption. */
        interface $Properties {

            /** GenderOption id */
            id?: (string|null);

            /** GenderOption label */
            label?: (string|null);

            /** GenderOption factionId */
            factionId?: (string|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a GenderOption. */
        type $Shape = game.GenderOption.$Properties;
    }

    /**
     * Properties of a GenderFaction.
     * @deprecated Use game.GenderFaction.$Properties instead.
     */
    interface IGenderFaction extends game.GenderFaction.$Properties {
    }

    /** Represents a GenderFaction. */
    class GenderFaction {

        /**
         * Constructs a new GenderFaction.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.GenderFaction.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** GenderFaction id. */
        id: string;

        /** GenderFaction label. */
        label: string;

        /** GenderFaction textColor. */
        textColor: string;

        /** GenderFaction backgroundColor. */
        backgroundColor: string;

        /** GenderFaction borderColor. */
        borderColor: string;

        /** GenderFaction taskGroup. */
        taskGroup: string;

        /**
         * Creates a new GenderFaction instance using the specified properties.
         * @param [properties] Properties to set
         * @returns GenderFaction instance
         */
        static create(properties: game.GenderFaction.$Shape): game.GenderFaction & game.GenderFaction.$Shape;
        static create(properties?: game.GenderFaction.$Properties): game.GenderFaction;

        /**
         * Encodes the specified GenderFaction message. Does not implicitly {@link game.GenderFaction.verify|verify} messages.
         * @param message GenderFaction message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.GenderFaction.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified GenderFaction message, length delimited. Does not implicitly {@link game.GenderFaction.verify|verify} messages.
         * @param message GenderFaction message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.GenderFaction.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a GenderFaction message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.GenderFaction & game.GenderFaction.$Shape} GenderFaction
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.GenderFaction & game.GenderFaction.$Shape;

        /**
         * Decodes a GenderFaction message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.GenderFaction & game.GenderFaction.$Shape} GenderFaction
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.GenderFaction & game.GenderFaction.$Shape;

        /**
         * Verifies a GenderFaction message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a GenderFaction message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns GenderFaction
         */
        static fromObject(object: { [k: string]: any }): game.GenderFaction;

        /**
         * Creates a plain object from a GenderFaction message. Also converts values to other types if specified.
         * @param message GenderFaction
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.GenderFaction, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this GenderFaction to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for GenderFaction
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace GenderFaction {

        /** Properties of a GenderFaction. */
        interface $Properties {

            /** GenderFaction id */
            id?: (string|null);

            /** GenderFaction label */
            label?: (string|null);

            /** GenderFaction textColor */
            textColor?: (string|null);

            /** GenderFaction backgroundColor */
            backgroundColor?: (string|null);

            /** GenderFaction borderColor */
            borderColor?: (string|null);

            /** GenderFaction taskGroup */
            taskGroup?: (string|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a GenderFaction. */
        type $Shape = game.GenderFaction.$Properties;
    }

    /**
     * Properties of a Pos.
     * @deprecated Use game.Pos.$Properties instead.
     */
    interface IPos extends game.Pos.$Properties {
    }

    /** Represents a Pos. */
    class Pos {

        /**
         * Constructs a new Pos.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.Pos.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** Pos row. */
        row: number;

        /** Pos col. */
        col: number;

        /**
         * Creates a new Pos instance using the specified properties.
         * @param [properties] Properties to set
         * @returns Pos instance
         */
        static create(properties: game.Pos.$Shape): game.Pos & game.Pos.$Shape;
        static create(properties?: game.Pos.$Properties): game.Pos;

        /**
         * Encodes the specified Pos message. Does not implicitly {@link game.Pos.verify|verify} messages.
         * @param message Pos message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.Pos.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified Pos message, length delimited. Does not implicitly {@link game.Pos.verify|verify} messages.
         * @param message Pos message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.Pos.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a Pos message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.Pos & game.Pos.$Shape} Pos
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.Pos & game.Pos.$Shape;

        /**
         * Decodes a Pos message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.Pos & game.Pos.$Shape} Pos
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.Pos & game.Pos.$Shape;

        /**
         * Verifies a Pos message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a Pos message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns Pos
         */
        static fromObject(object: { [k: string]: any }): game.Pos;

        /**
         * Creates a plain object from a Pos message. Also converts values to other types if specified.
         * @param message Pos
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.Pos, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this Pos to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for Pos
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace Pos {

        /** Properties of a Pos. */
        interface $Properties {

            /** Pos row */
            row?: (number|null);

            /** Pos col */
            col?: (number|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a Pos. */
        type $Shape = game.Pos.$Properties;
    }

    /**
     * Properties of a PublicStats.
     * @deprecated Use game.PublicStats.$Properties instead.
     */
    interface IPublicStats extends game.PublicStats.$Properties {
    }

    /** Represents a PublicStats. */
    class PublicStats {

        /**
         * Constructs a new PublicStats.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.PublicStats.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** PublicStats wins. */
        wins: number;

        /** PublicStats losses. */
        losses: number;

        /** PublicStats draws. */
        draws: number;

        /** PublicStats punishments. */
        punishments: number;

        /** PublicStats rankedPoints. */
        rankedPoints: number;

        /** PublicStats title. */
        title: string;

        /** PublicStats titleSegmentId. */
        titleSegmentId: string;

        /** PublicStats highestScore. */
        highestScore: number;

        /** PublicStats lowestScore. */
        lowestScore: number;

        /** PublicStats sortRankedPoints. */
        sortRankedPoints: number;

        /** PublicStats sortHighestScore. */
        sortHighestScore: number;

        /** PublicStats sortLowestScore. */
        sortLowestScore: number;

        /** PublicStats titleCustom. */
        titleCustom: boolean;

        /** PublicStats totalOnlineMs. */
        totalOnlineMs: (number|Long);

        /** PublicStats selfTitle. */
        selfTitle: string;

        /** PublicStats titleSource. */
        titleSource: string;

        /** PublicStats titleColors. */
        titleColors?: (game.GenderColors.$Properties|null);

        /**
         * Creates a new PublicStats instance using the specified properties.
         * @param [properties] Properties to set
         * @returns PublicStats instance
         */
        static create(properties: game.PublicStats.$Shape): game.PublicStats & game.PublicStats.$Shape;
        static create(properties?: game.PublicStats.$Properties): game.PublicStats;

        /**
         * Encodes the specified PublicStats message. Does not implicitly {@link game.PublicStats.verify|verify} messages.
         * @param message PublicStats message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.PublicStats.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified PublicStats message, length delimited. Does not implicitly {@link game.PublicStats.verify|verify} messages.
         * @param message PublicStats message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.PublicStats.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a PublicStats message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.PublicStats & game.PublicStats.$Shape} PublicStats
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.PublicStats & game.PublicStats.$Shape;

        /**
         * Decodes a PublicStats message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.PublicStats & game.PublicStats.$Shape} PublicStats
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.PublicStats & game.PublicStats.$Shape;

        /**
         * Verifies a PublicStats message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a PublicStats message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns PublicStats
         */
        static fromObject(object: { [k: string]: any }): game.PublicStats;

        /**
         * Creates a plain object from a PublicStats message. Also converts values to other types if specified.
         * @param message PublicStats
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.PublicStats, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this PublicStats to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for PublicStats
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace PublicStats {

        /** Properties of a PublicStats. */
        interface $Properties {

            /** PublicStats wins */
            wins?: (number|null);

            /** PublicStats losses */
            losses?: (number|null);

            /** PublicStats draws */
            draws?: (number|null);

            /** PublicStats punishments */
            punishments?: (number|null);

            /** PublicStats rankedPoints */
            rankedPoints?: (number|null);

            /** PublicStats title */
            title?: (string|null);

            /** PublicStats titleSegmentId */
            titleSegmentId?: (string|null);

            /** PublicStats highestScore */
            highestScore?: (number|null);

            /** PublicStats lowestScore */
            lowestScore?: (number|null);

            /** PublicStats sortRankedPoints */
            sortRankedPoints?: (number|null);

            /** PublicStats sortHighestScore */
            sortHighestScore?: (number|null);

            /** PublicStats sortLowestScore */
            sortLowestScore?: (number|null);

            /** PublicStats titleCustom */
            titleCustom?: (boolean|null);

            /** PublicStats totalOnlineMs */
            totalOnlineMs?: (number|Long|null);

            /** PublicStats selfTitle */
            selfTitle?: (string|null);

            /** PublicStats titleSource */
            titleSource?: (string|null);

            /** PublicStats titleColors */
            titleColors?: (game.GenderColors.$Properties|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a PublicStats. */
        type $Shape = game.PublicStats.$Properties;
    }

    /**
     * Properties of a LobbyStats.
     * @deprecated Use game.LobbyStats.$Properties instead.
     */
    interface ILobbyStats extends game.LobbyStats.$Properties {
    }

    /** Represents a LobbyStats. */
    class LobbyStats {

        /**
         * Constructs a new LobbyStats.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.LobbyStats.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** LobbyStats wins. */
        wins: number;

        /** LobbyStats losses. */
        losses: number;

        /** LobbyStats draws. */
        draws: number;

        /** LobbyStats punishments. */
        punishments: number;

        /** LobbyStats rankedPoints. */
        rankedPoints: number;

        /** LobbyStats title. */
        title: string;

        /** LobbyStats highestScore. */
        highestScore: number;

        /** LobbyStats lowestScore. */
        lowestScore: number;

        /** LobbyStats sortRankedPoints. */
        sortRankedPoints: number;

        /** LobbyStats sortHighestScore. */
        sortHighestScore: number;

        /** LobbyStats sortLowestScore. */
        sortLowestScore: number;

        /** LobbyStats titleCustom. */
        titleCustom: boolean;

        /** LobbyStats totalOnlineMs. */
        totalOnlineMs: (number|Long);

        /** LobbyStats selfTitle. */
        selfTitle: string;

        /** LobbyStats titleSource. */
        titleSource: string;

        /** LobbyStats titleColors. */
        titleColors?: (game.GenderColors.$Properties|null);

        /**
         * Creates a new LobbyStats instance using the specified properties.
         * @param [properties] Properties to set
         * @returns LobbyStats instance
         */
        static create(properties: game.LobbyStats.$Shape): game.LobbyStats & game.LobbyStats.$Shape;
        static create(properties?: game.LobbyStats.$Properties): game.LobbyStats;

        /**
         * Encodes the specified LobbyStats message. Does not implicitly {@link game.LobbyStats.verify|verify} messages.
         * @param message LobbyStats message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.LobbyStats.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified LobbyStats message, length delimited. Does not implicitly {@link game.LobbyStats.verify|verify} messages.
         * @param message LobbyStats message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.LobbyStats.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a LobbyStats message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.LobbyStats & game.LobbyStats.$Shape} LobbyStats
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.LobbyStats & game.LobbyStats.$Shape;

        /**
         * Decodes a LobbyStats message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.LobbyStats & game.LobbyStats.$Shape} LobbyStats
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.LobbyStats & game.LobbyStats.$Shape;

        /**
         * Verifies a LobbyStats message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a LobbyStats message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns LobbyStats
         */
        static fromObject(object: { [k: string]: any }): game.LobbyStats;

        /**
         * Creates a plain object from a LobbyStats message. Also converts values to other types if specified.
         * @param message LobbyStats
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.LobbyStats, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this LobbyStats to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for LobbyStats
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace LobbyStats {

        /** Properties of a LobbyStats. */
        interface $Properties {

            /** LobbyStats wins */
            wins?: (number|null);

            /** LobbyStats losses */
            losses?: (number|null);

            /** LobbyStats draws */
            draws?: (number|null);

            /** LobbyStats punishments */
            punishments?: (number|null);

            /** LobbyStats rankedPoints */
            rankedPoints?: (number|null);

            /** LobbyStats title */
            title?: (string|null);

            /** LobbyStats highestScore */
            highestScore?: (number|null);

            /** LobbyStats lowestScore */
            lowestScore?: (number|null);

            /** LobbyStats sortRankedPoints */
            sortRankedPoints?: (number|null);

            /** LobbyStats sortHighestScore */
            sortHighestScore?: (number|null);

            /** LobbyStats sortLowestScore */
            sortLowestScore?: (number|null);

            /** LobbyStats titleCustom */
            titleCustom?: (boolean|null);

            /** LobbyStats totalOnlineMs */
            totalOnlineMs?: (number|Long|null);

            /** LobbyStats selfTitle */
            selfTitle?: (string|null);

            /** LobbyStats titleSource */
            titleSource?: (string|null);

            /** LobbyStats titleColors */
            titleColors?: (game.GenderColors.$Properties|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a LobbyStats. */
        type $Shape = game.LobbyStats.$Properties;
    }

    /**
     * Properties of a GameWLD.
     * @deprecated Use game.GameWLD.$Properties instead.
     */
    interface IGameWLD extends game.GameWLD.$Properties {
    }

    /** Represents a GameWLD. */
    class GameWLD {

        /**
         * Constructs a new GameWLD.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.GameWLD.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** GameWLD wins. */
        wins: number;

        /** GameWLD losses. */
        losses: number;

        /** GameWLD draws. */
        draws: number;

        /**
         * Creates a new GameWLD instance using the specified properties.
         * @param [properties] Properties to set
         * @returns GameWLD instance
         */
        static create(properties: game.GameWLD.$Shape): game.GameWLD & game.GameWLD.$Shape;
        static create(properties?: game.GameWLD.$Properties): game.GameWLD;

        /**
         * Encodes the specified GameWLD message. Does not implicitly {@link game.GameWLD.verify|verify} messages.
         * @param message GameWLD message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.GameWLD.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified GameWLD message, length delimited. Does not implicitly {@link game.GameWLD.verify|verify} messages.
         * @param message GameWLD message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.GameWLD.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a GameWLD message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.GameWLD & game.GameWLD.$Shape} GameWLD
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.GameWLD & game.GameWLD.$Shape;

        /**
         * Decodes a GameWLD message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.GameWLD & game.GameWLD.$Shape} GameWLD
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.GameWLD & game.GameWLD.$Shape;

        /**
         * Verifies a GameWLD message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a GameWLD message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns GameWLD
         */
        static fromObject(object: { [k: string]: any }): game.GameWLD;

        /**
         * Creates a plain object from a GameWLD message. Also converts values to other types if specified.
         * @param message GameWLD
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.GameWLD, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this GameWLD to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for GameWLD
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace GameWLD {

        /** Properties of a GameWLD. */
        interface $Properties {

            /** GameWLD wins */
            wins?: (number|null);

            /** GameWLD losses */
            losses?: (number|null);

            /** GameWLD draws */
            draws?: (number|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a GameWLD. */
        type $Shape = game.GameWLD.$Properties;
    }

    /**
     * Properties of a GameStats.
     * @deprecated Use game.GameStats.$Properties instead.
     */
    interface IGameStats extends game.GameStats.$Properties {
    }

    /** Represents a GameStats. */
    class GameStats {

        /**
         * Constructs a new GameStats.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.GameStats.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** GameStats rps. */
        rps?: (game.GameWLD.$Properties|null);

        /** GameStats othello. */
        othello?: (game.GameWLD.$Properties|null);

        /** GameStats tictactoe. */
        tictactoe?: (game.GameWLD.$Properties|null);

        /** GameStats gomoku. */
        gomoku?: (game.GameWLD.$Properties|null);

        /** GameStats liarsdice. */
        liarsdice?: (game.GameWLD.$Properties|null);

        /** GameStats jungle. */
        jungle?: (game.GameWLD.$Properties|null);

        /** GameStats chess. */
        chess?: (game.GameWLD.$Properties|null);

        /**
         * Creates a new GameStats instance using the specified properties.
         * @param [properties] Properties to set
         * @returns GameStats instance
         */
        static create(properties: game.GameStats.$Shape): game.GameStats & game.GameStats.$Shape;
        static create(properties?: game.GameStats.$Properties): game.GameStats;

        /**
         * Encodes the specified GameStats message. Does not implicitly {@link game.GameStats.verify|verify} messages.
         * @param message GameStats message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.GameStats.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified GameStats message, length delimited. Does not implicitly {@link game.GameStats.verify|verify} messages.
         * @param message GameStats message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.GameStats.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a GameStats message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.GameStats & game.GameStats.$Shape} GameStats
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.GameStats & game.GameStats.$Shape;

        /**
         * Decodes a GameStats message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.GameStats & game.GameStats.$Shape} GameStats
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.GameStats & game.GameStats.$Shape;

        /**
         * Verifies a GameStats message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a GameStats message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns GameStats
         */
        static fromObject(object: { [k: string]: any }): game.GameStats;

        /**
         * Creates a plain object from a GameStats message. Also converts values to other types if specified.
         * @param message GameStats
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.GameStats, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this GameStats to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for GameStats
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace GameStats {

        /** Properties of a GameStats. */
        interface $Properties {

            /** GameStats rps */
            rps?: (game.GameWLD.$Properties|null);

            /** GameStats othello */
            othello?: (game.GameWLD.$Properties|null);

            /** GameStats tictactoe */
            tictactoe?: (game.GameWLD.$Properties|null);

            /** GameStats gomoku */
            gomoku?: (game.GameWLD.$Properties|null);

            /** GameStats liarsdice */
            liarsdice?: (game.GameWLD.$Properties|null);

            /** GameStats jungle */
            jungle?: (game.GameWLD.$Properties|null);

            /** GameStats chess */
            chess?: (game.GameWLD.$Properties|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a GameStats. */
        type $Shape = game.GameStats.$Properties;
    }

    /**
     * Properties of a PublicPlayer.
     * @deprecated Use game.PublicPlayer.$Properties instead.
     */
    interface IPublicPlayer extends game.PublicPlayer.$Properties {
    }

    /** Represents a PublicPlayer. */
    class PublicPlayer {

        /**
         * Constructs a new PublicPlayer.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.PublicPlayer.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** PublicPlayer id. */
        id: string;

        /** PublicPlayer name. */
        name: string;

        /** PublicPlayer genderId. */
        genderId: string;

        /** PublicPlayer genderLabel. */
        genderLabel: string;

        /** PublicPlayer factionId. */
        factionId: string;

        /** PublicPlayer avatarUrl. */
        avatarUrl: string;

        /** PublicPlayer factionLabel. */
        factionLabel: string;

        /** PublicPlayer factionColors. */
        factionColors?: (game.GenderColors.$Properties|null);

        /** PublicPlayer displayName. */
        displayName: string;

        /** PublicPlayer connected. */
        connected: boolean;

        /** PublicPlayer disconnectedAt. */
        disconnectedAt: (number|Long);

        /** PublicPlayer disconnectExpiresAt. */
        disconnectExpiresAt: (number|Long);

        /** PublicPlayer profileUpdatedAt. */
        profileUpdatedAt: (number|Long);

        /** PublicPlayer nameWarEnabled. */
        nameWarEnabled: boolean;

        /** PublicPlayer nameWarToggledAt. */
        nameWarToggledAt: (number|Long);

        /** PublicPlayer nameWarOriginalName. */
        nameWarOriginalName: string;

        /** PublicPlayer nameWarPenaltyName. */
        nameWarPenaltyName: string;

        /** PublicPlayer nameWarPunished. */
        nameWarPunished: boolean;

        /** PublicPlayer nameWarAllowRename. */
        nameWarAllowRename: boolean;

        /** PublicPlayer nameWarRenameProtectedUntil. */
        nameWarRenameProtectedUntil: (number|Long);

        /** PublicPlayer nameWarRenamedBy. */
        nameWarRenamedBy: string;

        /** PublicPlayer nameWarRenamedByName. */
        nameWarRenamedByName: string;

        /** PublicPlayer nameWarRenameWindowStartedAt. */
        nameWarRenameWindowStartedAt: (number|Long);

        /** PublicPlayer nameWarRenameCount. */
        nameWarRenameCount: number;

        /** PublicPlayer giveawayEnabled. */
        giveawayEnabled: boolean;

        /** PublicPlayer giveawayValue. */
        giveawayValue: number;

        /** PublicPlayer giveawayClicks. */
        giveawayClicks: number;

        /** PublicPlayer giveawayBoardText. */
        giveawayBoardText: string;

        /** PublicPlayer giveawayBoardSubmittedAt. */
        giveawayBoardSubmittedAt: (number|Long);

        /** PublicPlayer giveawayBoardExpiresAt. */
        giveawayBoardExpiresAt: (number|Long);

        /** PublicPlayer giveawayBoardLikes. */
        giveawayBoardLikes: number;

        /** PublicPlayer giveawayBoardDislikes. */
        giveawayBoardDislikes: number;

        /** PublicPlayer giveawayBoardLikesThisHour. */
        giveawayBoardLikesThisHour: number;

        /** PublicPlayer giveawayBoardLikeWindowStartedAt. */
        giveawayBoardLikeWindowStartedAt: (number|Long);

        /** PublicPlayer giveawayVoteWindowStartedAt. */
        giveawayVoteWindowStartedAt: (number|Long);

        /** PublicPlayer giveawayVoteCount. */
        giveawayVoteCount: number;

        /** PublicPlayer giveawayVoteLikesThisHour. */
        giveawayVoteLikesThisHour: number;

        /** PublicPlayer giveawayVoteDislikesThisHour. */
        giveawayVoteDislikesThisHour: number;

        /** PublicPlayer rankMultiplierUnlocked. */
        rankMultiplierUnlocked: boolean;

        /** PublicPlayer extremeModeEnabled. */
        extremeModeEnabled: boolean;

        /** PublicPlayer extremeModeToggledAt. */
        extremeModeToggledAt: (number|Long);

        /** PublicPlayer extremeModeCooldownUntil. */
        extremeModeCooldownUntil: (number|Long);

        /** PublicPlayer extremeWinStreak. */
        extremeWinStreak: number;

        /** PublicPlayer extremeLastDecayHour. */
        extremeLastDecayHour: (number|Long);

        /** PublicPlayer extremeForceClosed. */
        extremeForceClosed: boolean;

        /** PublicPlayer extremeForceClosedAt. */
        extremeForceClosedAt: (number|Long);

        /** PublicPlayer extremeRenameProtectedUntil. */
        extremeRenameProtectedUntil: (number|Long);

        /** PublicPlayer extremeRenamedBy. */
        extremeRenamedBy: string;

        /** PublicPlayer extremeRenamedByName. */
        extremeRenamedByName: string;

        /** PublicPlayer roomId. */
        roomId: string;

        /** PublicPlayer isAdmin. */
        isAdmin: boolean;

        /** PublicPlayer stats. */
        stats?: (game.PublicStats.$Properties|null);

        /** PublicPlayer gameStats. */
        gameStats?: (game.GameStats.$Properties|null);

        /** PublicPlayer bondMasterEnabled. */
        bondMasterEnabled: boolean;

        /** PublicPlayer bondPetEnabled. */
        bondPetEnabled: boolean;

        /** PublicPlayer bondPublicDisplay. */
        bondPublicDisplay: boolean;

        /**
         * Creates a new PublicPlayer instance using the specified properties.
         * @param [properties] Properties to set
         * @returns PublicPlayer instance
         */
        static create(properties: game.PublicPlayer.$Shape): game.PublicPlayer & game.PublicPlayer.$Shape;
        static create(properties?: game.PublicPlayer.$Properties): game.PublicPlayer;

        /**
         * Encodes the specified PublicPlayer message. Does not implicitly {@link game.PublicPlayer.verify|verify} messages.
         * @param message PublicPlayer message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.PublicPlayer.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified PublicPlayer message, length delimited. Does not implicitly {@link game.PublicPlayer.verify|verify} messages.
         * @param message PublicPlayer message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.PublicPlayer.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a PublicPlayer message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.PublicPlayer & game.PublicPlayer.$Shape} PublicPlayer
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.PublicPlayer & game.PublicPlayer.$Shape;

        /**
         * Decodes a PublicPlayer message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.PublicPlayer & game.PublicPlayer.$Shape} PublicPlayer
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.PublicPlayer & game.PublicPlayer.$Shape;

        /**
         * Verifies a PublicPlayer message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a PublicPlayer message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns PublicPlayer
         */
        static fromObject(object: { [k: string]: any }): game.PublicPlayer;

        /**
         * Creates a plain object from a PublicPlayer message. Also converts values to other types if specified.
         * @param message PublicPlayer
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.PublicPlayer, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this PublicPlayer to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for PublicPlayer
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace PublicPlayer {

        /** Properties of a PublicPlayer. */
        interface $Properties {

            /** PublicPlayer id */
            id?: (string|null);

            /** PublicPlayer name */
            name?: (string|null);

            /** PublicPlayer genderId */
            genderId?: (string|null);

            /** PublicPlayer genderLabel */
            genderLabel?: (string|null);

            /** PublicPlayer factionId */
            factionId?: (string|null);

            /** PublicPlayer avatarUrl */
            avatarUrl?: (string|null);

            /** PublicPlayer factionLabel */
            factionLabel?: (string|null);

            /** PublicPlayer factionColors */
            factionColors?: (game.GenderColors.$Properties|null);

            /** PublicPlayer displayName */
            displayName?: (string|null);

            /** PublicPlayer connected */
            connected?: (boolean|null);

            /** PublicPlayer disconnectedAt */
            disconnectedAt?: (number|Long|null);

            /** PublicPlayer disconnectExpiresAt */
            disconnectExpiresAt?: (number|Long|null);

            /** PublicPlayer profileUpdatedAt */
            profileUpdatedAt?: (number|Long|null);

            /** PublicPlayer nameWarEnabled */
            nameWarEnabled?: (boolean|null);

            /** PublicPlayer nameWarToggledAt */
            nameWarToggledAt?: (number|Long|null);

            /** PublicPlayer nameWarOriginalName */
            nameWarOriginalName?: (string|null);

            /** PublicPlayer nameWarPenaltyName */
            nameWarPenaltyName?: (string|null);

            /** PublicPlayer nameWarPunished */
            nameWarPunished?: (boolean|null);

            /** PublicPlayer nameWarAllowRename */
            nameWarAllowRename?: (boolean|null);

            /** PublicPlayer nameWarRenameProtectedUntil */
            nameWarRenameProtectedUntil?: (number|Long|null);

            /** PublicPlayer nameWarRenamedBy */
            nameWarRenamedBy?: (string|null);

            /** PublicPlayer nameWarRenamedByName */
            nameWarRenamedByName?: (string|null);

            /** PublicPlayer nameWarRenameWindowStartedAt */
            nameWarRenameWindowStartedAt?: (number|Long|null);

            /** PublicPlayer nameWarRenameCount */
            nameWarRenameCount?: (number|null);

            /** PublicPlayer giveawayEnabled */
            giveawayEnabled?: (boolean|null);

            /** PublicPlayer giveawayValue */
            giveawayValue?: (number|null);

            /** PublicPlayer giveawayClicks */
            giveawayClicks?: (number|null);

            /** PublicPlayer giveawayBoardText */
            giveawayBoardText?: (string|null);

            /** PublicPlayer giveawayBoardSubmittedAt */
            giveawayBoardSubmittedAt?: (number|Long|null);

            /** PublicPlayer giveawayBoardExpiresAt */
            giveawayBoardExpiresAt?: (number|Long|null);

            /** PublicPlayer giveawayBoardLikes */
            giveawayBoardLikes?: (number|null);

            /** PublicPlayer giveawayBoardDislikes */
            giveawayBoardDislikes?: (number|null);

            /** PublicPlayer giveawayBoardLikesThisHour */
            giveawayBoardLikesThisHour?: (number|null);

            /** PublicPlayer giveawayBoardLikeWindowStartedAt */
            giveawayBoardLikeWindowStartedAt?: (number|Long|null);

            /** PublicPlayer giveawayVoteWindowStartedAt */
            giveawayVoteWindowStartedAt?: (number|Long|null);

            /** PublicPlayer giveawayVoteCount */
            giveawayVoteCount?: (number|null);

            /** PublicPlayer giveawayVoteLikesThisHour */
            giveawayVoteLikesThisHour?: (number|null);

            /** PublicPlayer giveawayVoteDislikesThisHour */
            giveawayVoteDislikesThisHour?: (number|null);

            /** PublicPlayer rankMultiplierUnlocked */
            rankMultiplierUnlocked?: (boolean|null);

            /** PublicPlayer extremeModeEnabled */
            extremeModeEnabled?: (boolean|null);

            /** PublicPlayer extremeModeToggledAt */
            extremeModeToggledAt?: (number|Long|null);

            /** PublicPlayer extremeModeCooldownUntil */
            extremeModeCooldownUntil?: (number|Long|null);

            /** PublicPlayer extremeWinStreak */
            extremeWinStreak?: (number|null);

            /** PublicPlayer extremeLastDecayHour */
            extremeLastDecayHour?: (number|Long|null);

            /** PublicPlayer extremeForceClosed */
            extremeForceClosed?: (boolean|null);

            /** PublicPlayer extremeForceClosedAt */
            extremeForceClosedAt?: (number|Long|null);

            /** PublicPlayer extremeRenameProtectedUntil */
            extremeRenameProtectedUntil?: (number|Long|null);

            /** PublicPlayer extremeRenamedBy */
            extremeRenamedBy?: (string|null);

            /** PublicPlayer extremeRenamedByName */
            extremeRenamedByName?: (string|null);

            /** PublicPlayer roomId */
            roomId?: (string|null);

            /** PublicPlayer isAdmin */
            isAdmin?: (boolean|null);

            /** PublicPlayer stats */
            stats?: (game.PublicStats.$Properties|null);

            /** PublicPlayer gameStats */
            gameStats?: (game.GameStats.$Properties|null);

            /** PublicPlayer bondMasterEnabled */
            bondMasterEnabled?: (boolean|null);

            /** PublicPlayer bondPetEnabled */
            bondPetEnabled?: (boolean|null);

            /** PublicPlayer bondPublicDisplay */
            bondPublicDisplay?: (boolean|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a PublicPlayer. */
        type $Shape = game.PublicPlayer.$Properties;
    }

    /**
     * Properties of a LobbyPlayer.
     * @deprecated Use game.LobbyPlayer.$Properties instead.
     */
    interface ILobbyPlayer extends game.LobbyPlayer.$Properties {
    }

    /** Represents a LobbyPlayer. */
    class LobbyPlayer {

        /**
         * Constructs a new LobbyPlayer.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.LobbyPlayer.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** LobbyPlayer id. */
        id: string;

        /** LobbyPlayer name. */
        name: string;

        /** LobbyPlayer genderId. */
        genderId: string;

        /** LobbyPlayer genderLabel. */
        genderLabel: string;

        /** LobbyPlayer factionId. */
        factionId: string;

        /** LobbyPlayer factionLabel. */
        factionLabel: string;

        /** LobbyPlayer factionColors. */
        factionColors?: (game.GenderColors.$Properties|null);

        /** LobbyPlayer displayName. */
        displayName: string;

        /** LobbyPlayer connected. */
        connected: boolean;

        /** LobbyPlayer disconnectedAt. */
        disconnectedAt: (number|Long);

        /** LobbyPlayer disconnectExpiresAt. */
        disconnectExpiresAt: (number|Long);

        /** LobbyPlayer nameWarEnabled. */
        nameWarEnabled: boolean;

        /** LobbyPlayer nameWarPenaltyName. */
        nameWarPenaltyName: string;

        /** LobbyPlayer nameWarPunished. */
        nameWarPunished: boolean;

        /** LobbyPlayer nameWarAllowRename. */
        nameWarAllowRename: boolean;

        /** LobbyPlayer nameWarRenameProtectedUntil. */
        nameWarRenameProtectedUntil: (number|Long);

        /** LobbyPlayer nameWarRenamedByName. */
        nameWarRenamedByName: string;

        /** LobbyPlayer giveawayEnabled. */
        giveawayEnabled: boolean;

        /** LobbyPlayer giveawayValue. */
        giveawayValue: number;

        /** LobbyPlayer giveawayBoardText. */
        giveawayBoardText: string;

        /** LobbyPlayer giveawayBoardExpiresAt. */
        giveawayBoardExpiresAt: (number|Long);

        /** LobbyPlayer giveawayBoardLikes. */
        giveawayBoardLikes: number;

        /** LobbyPlayer giveawayBoardDislikes. */
        giveawayBoardDislikes: number;

        /** LobbyPlayer giveawayVoteWindowStartedAt. */
        giveawayVoteWindowStartedAt: (number|Long);

        /** LobbyPlayer giveawayVoteLikesThisHour. */
        giveawayVoteLikesThisHour: number;

        /** LobbyPlayer giveawayVoteDislikesThisHour. */
        giveawayVoteDislikesThisHour: number;

        /** LobbyPlayer rankMultiplierUnlocked. */
        rankMultiplierUnlocked: boolean;

        /** LobbyPlayer extremeModeEnabled. */
        extremeModeEnabled: boolean;

        /** LobbyPlayer extremeWinStreak. */
        extremeWinStreak: number;

        /** LobbyPlayer extremeForceClosed. */
        extremeForceClosed: boolean;

        /** LobbyPlayer extremeForceClosedAt. */
        extremeForceClosedAt: (number|Long);

        /** LobbyPlayer extremeRenameProtectedUntil. */
        extremeRenameProtectedUntil: (number|Long);

        /** LobbyPlayer extremeRenamedByName. */
        extremeRenamedByName: string;

        /** LobbyPlayer roomId. */
        roomId: string;

        /** LobbyPlayer stats. */
        stats?: (game.LobbyStats.$Properties|null);

        /** LobbyPlayer gameStats. */
        gameStats?: (game.GameStats.$Properties|null);

        /** LobbyPlayer avatarUrl. */
        avatarUrl: string;

        /** LobbyPlayer bondMasterEnabled. */
        bondMasterEnabled: boolean;

        /** LobbyPlayer bondPetEnabled. */
        bondPetEnabled: boolean;

        /** LobbyPlayer bondPublicDisplay. */
        bondPublicDisplay: boolean;

        /**
         * Creates a new LobbyPlayer instance using the specified properties.
         * @param [properties] Properties to set
         * @returns LobbyPlayer instance
         */
        static create(properties: game.LobbyPlayer.$Shape): game.LobbyPlayer & game.LobbyPlayer.$Shape;
        static create(properties?: game.LobbyPlayer.$Properties): game.LobbyPlayer;

        /**
         * Encodes the specified LobbyPlayer message. Does not implicitly {@link game.LobbyPlayer.verify|verify} messages.
         * @param message LobbyPlayer message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.LobbyPlayer.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified LobbyPlayer message, length delimited. Does not implicitly {@link game.LobbyPlayer.verify|verify} messages.
         * @param message LobbyPlayer message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.LobbyPlayer.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a LobbyPlayer message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.LobbyPlayer & game.LobbyPlayer.$Shape} LobbyPlayer
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.LobbyPlayer & game.LobbyPlayer.$Shape;

        /**
         * Decodes a LobbyPlayer message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.LobbyPlayer & game.LobbyPlayer.$Shape} LobbyPlayer
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.LobbyPlayer & game.LobbyPlayer.$Shape;

        /**
         * Verifies a LobbyPlayer message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a LobbyPlayer message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns LobbyPlayer
         */
        static fromObject(object: { [k: string]: any }): game.LobbyPlayer;

        /**
         * Creates a plain object from a LobbyPlayer message. Also converts values to other types if specified.
         * @param message LobbyPlayer
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.LobbyPlayer, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this LobbyPlayer to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for LobbyPlayer
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace LobbyPlayer {

        /** Properties of a LobbyPlayer. */
        interface $Properties {

            /** LobbyPlayer id */
            id?: (string|null);

            /** LobbyPlayer name */
            name?: (string|null);

            /** LobbyPlayer genderId */
            genderId?: (string|null);

            /** LobbyPlayer genderLabel */
            genderLabel?: (string|null);

            /** LobbyPlayer factionId */
            factionId?: (string|null);

            /** LobbyPlayer factionLabel */
            factionLabel?: (string|null);

            /** LobbyPlayer factionColors */
            factionColors?: (game.GenderColors.$Properties|null);

            /** LobbyPlayer displayName */
            displayName?: (string|null);

            /** LobbyPlayer connected */
            connected?: (boolean|null);

            /** LobbyPlayer disconnectedAt */
            disconnectedAt?: (number|Long|null);

            /** LobbyPlayer disconnectExpiresAt */
            disconnectExpiresAt?: (number|Long|null);

            /** LobbyPlayer nameWarEnabled */
            nameWarEnabled?: (boolean|null);

            /** LobbyPlayer nameWarPenaltyName */
            nameWarPenaltyName?: (string|null);

            /** LobbyPlayer nameWarPunished */
            nameWarPunished?: (boolean|null);

            /** LobbyPlayer nameWarAllowRename */
            nameWarAllowRename?: (boolean|null);

            /** LobbyPlayer nameWarRenameProtectedUntil */
            nameWarRenameProtectedUntil?: (number|Long|null);

            /** LobbyPlayer nameWarRenamedByName */
            nameWarRenamedByName?: (string|null);

            /** LobbyPlayer giveawayEnabled */
            giveawayEnabled?: (boolean|null);

            /** LobbyPlayer giveawayValue */
            giveawayValue?: (number|null);

            /** LobbyPlayer giveawayBoardText */
            giveawayBoardText?: (string|null);

            /** LobbyPlayer giveawayBoardExpiresAt */
            giveawayBoardExpiresAt?: (number|Long|null);

            /** LobbyPlayer giveawayBoardLikes */
            giveawayBoardLikes?: (number|null);

            /** LobbyPlayer giveawayBoardDislikes */
            giveawayBoardDislikes?: (number|null);

            /** LobbyPlayer giveawayVoteWindowStartedAt */
            giveawayVoteWindowStartedAt?: (number|Long|null);

            /** LobbyPlayer giveawayVoteLikesThisHour */
            giveawayVoteLikesThisHour?: (number|null);

            /** LobbyPlayer giveawayVoteDislikesThisHour */
            giveawayVoteDislikesThisHour?: (number|null);

            /** LobbyPlayer rankMultiplierUnlocked */
            rankMultiplierUnlocked?: (boolean|null);

            /** LobbyPlayer extremeModeEnabled */
            extremeModeEnabled?: (boolean|null);

            /** LobbyPlayer extremeWinStreak */
            extremeWinStreak?: (number|null);

            /** LobbyPlayer extremeForceClosed */
            extremeForceClosed?: (boolean|null);

            /** LobbyPlayer extremeForceClosedAt */
            extremeForceClosedAt?: (number|Long|null);

            /** LobbyPlayer extremeRenameProtectedUntil */
            extremeRenameProtectedUntil?: (number|Long|null);

            /** LobbyPlayer extremeRenamedByName */
            extremeRenamedByName?: (string|null);

            /** LobbyPlayer roomId */
            roomId?: (string|null);

            /** LobbyPlayer stats */
            stats?: (game.LobbyStats.$Properties|null);

            /** LobbyPlayer gameStats */
            gameStats?: (game.GameStats.$Properties|null);

            /** LobbyPlayer avatarUrl */
            avatarUrl?: (string|null);

            /** LobbyPlayer bondMasterEnabled */
            bondMasterEnabled?: (boolean|null);

            /** LobbyPlayer bondPetEnabled */
            bondPetEnabled?: (boolean|null);

            /** LobbyPlayer bondPublicDisplay */
            bondPublicDisplay?: (boolean|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a LobbyPlayer. */
        type $Shape = game.LobbyPlayer.$Properties;
    }

    /**
     * Properties of a SeatOccupant.
     * @deprecated Use game.SeatOccupant.$Properties instead.
     */
    interface ISeatOccupant extends game.SeatOccupant.$Properties {
    }

    /** Represents a SeatOccupant. */
    class SeatOccupant {

        /**
         * Constructs a new SeatOccupant.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.SeatOccupant.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** SeatOccupant player. */
        player?: (game.PublicPlayer.$Properties|null);

        /** SeatOccupant kind. */
        kind?: "player";

        /**
         * Creates a new SeatOccupant instance using the specified properties.
         * @param [properties] Properties to set
         * @returns SeatOccupant instance
         */
        static create(properties: game.SeatOccupant.$Shape): game.SeatOccupant & game.SeatOccupant.$Shape;
        static create(properties?: game.SeatOccupant.$Properties): game.SeatOccupant;

        /**
         * Encodes the specified SeatOccupant message. Does not implicitly {@link game.SeatOccupant.verify|verify} messages.
         * @param message SeatOccupant message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.SeatOccupant.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified SeatOccupant message, length delimited. Does not implicitly {@link game.SeatOccupant.verify|verify} messages.
         * @param message SeatOccupant message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.SeatOccupant.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a SeatOccupant message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.SeatOccupant & game.SeatOccupant.$Shape} SeatOccupant
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.SeatOccupant & game.SeatOccupant.$Shape;

        /**
         * Decodes a SeatOccupant message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.SeatOccupant & game.SeatOccupant.$Shape} SeatOccupant
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.SeatOccupant & game.SeatOccupant.$Shape;

        /**
         * Verifies a SeatOccupant message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a SeatOccupant message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns SeatOccupant
         */
        static fromObject(object: { [k: string]: any }): game.SeatOccupant;

        /**
         * Creates a plain object from a SeatOccupant message. Also converts values to other types if specified.
         * @param message SeatOccupant
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.SeatOccupant, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this SeatOccupant to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for SeatOccupant
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace SeatOccupant {

        /** Properties of a SeatOccupant. */
        interface $Properties {

            /** SeatOccupant player */
            player?: (game.PublicPlayer.$Properties|null);

            /** SeatOccupant kind */
            kind?: "player";

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Narrowed shape of a SeatOccupant. */
        type $Shape = {
          player?: game.PublicPlayer.$Shape|null;
          $unknowns?: Uint8Array[];
        } & (
          ({ kind?: undefined; player?: null }|{ kind?: "player"; player: game.PublicPlayer.$Shape })
        );
    }

    /**
     * Properties of a ChatMessage.
     * @deprecated Use game.ChatMessage.$Properties instead.
     */
    interface IChatMessage extends game.ChatMessage.$Properties {
    }

    /** Represents a ChatMessage. */
    class ChatMessage {

        /**
         * Constructs a new ChatMessage.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.ChatMessage.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** ChatMessage id. */
        id: string;

        /** ChatMessage roomId. */
        roomId: string;

        /** ChatMessage playerId. */
        playerId: string;

        /** ChatMessage author. */
        author: string;

        /** ChatMessage authorRole. */
        authorRole: string;

        /** ChatMessage text. */
        text: string;

        /** ChatMessage at. */
        at: (number|Long);

        /** ChatMessage system. */
        system: boolean;

        /** ChatMessage transient. */
        transient: boolean;

        /** ChatMessage expiresAt. */
        expiresAt: (number|Long);

        /** ChatMessage mentions. */
        mentions: string[];

        /** ChatMessage seq. */
        seq: (number|Long);

        /**
         * Creates a new ChatMessage instance using the specified properties.
         * @param [properties] Properties to set
         * @returns ChatMessage instance
         */
        static create(properties: game.ChatMessage.$Shape): game.ChatMessage & game.ChatMessage.$Shape;
        static create(properties?: game.ChatMessage.$Properties): game.ChatMessage;

        /**
         * Encodes the specified ChatMessage message. Does not implicitly {@link game.ChatMessage.verify|verify} messages.
         * @param message ChatMessage message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.ChatMessage.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified ChatMessage message, length delimited. Does not implicitly {@link game.ChatMessage.verify|verify} messages.
         * @param message ChatMessage message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.ChatMessage.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a ChatMessage message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.ChatMessage & game.ChatMessage.$Shape} ChatMessage
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.ChatMessage & game.ChatMessage.$Shape;

        /**
         * Decodes a ChatMessage message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.ChatMessage & game.ChatMessage.$Shape} ChatMessage
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.ChatMessage & game.ChatMessage.$Shape;

        /**
         * Verifies a ChatMessage message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a ChatMessage message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns ChatMessage
         */
        static fromObject(object: { [k: string]: any }): game.ChatMessage;

        /**
         * Creates a plain object from a ChatMessage message. Also converts values to other types if specified.
         * @param message ChatMessage
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.ChatMessage, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this ChatMessage to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for ChatMessage
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace ChatMessage {

        /** Properties of a ChatMessage. */
        interface $Properties {

            /** ChatMessage id */
            id?: (string|null);

            /** ChatMessage roomId */
            roomId?: (string|null);

            /** ChatMessage playerId */
            playerId?: (string|null);

            /** ChatMessage author */
            author?: (string|null);

            /** ChatMessage authorRole */
            authorRole?: (string|null);

            /** ChatMessage text */
            text?: (string|null);

            /** ChatMessage at */
            at?: (number|Long|null);

            /** ChatMessage system */
            system?: (boolean|null);

            /** ChatMessage transient */
            transient?: (boolean|null);

            /** ChatMessage expiresAt */
            expiresAt?: (number|Long|null);

            /** ChatMessage mentions */
            mentions?: (string[]|null);

            /** ChatMessage seq */
            seq?: (number|Long|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a ChatMessage. */
        type $Shape = game.ChatMessage.$Properties;
    }

    /**
     * Properties of a Suggestion.
     * @deprecated Use game.Suggestion.$Properties instead.
     */
    interface ISuggestion extends game.Suggestion.$Properties {
    }

    /** Represents a Suggestion. */
    class Suggestion {

        /**
         * Constructs a new Suggestion.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.Suggestion.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** Suggestion id. */
        id: string;

        /** Suggestion playerId. */
        playerId: string;

        /** Suggestion author. */
        author: string;

        /** Suggestion authorPlayer. */
        authorPlayer?: (game.PublicPlayer.$Properties|null);

        /** Suggestion text. */
        text: string;

        /** Suggestion at. */
        at: (number|Long);

        /**
         * Creates a new Suggestion instance using the specified properties.
         * @param [properties] Properties to set
         * @returns Suggestion instance
         */
        static create(properties: game.Suggestion.$Shape): game.Suggestion & game.Suggestion.$Shape;
        static create(properties?: game.Suggestion.$Properties): game.Suggestion;

        /**
         * Encodes the specified Suggestion message. Does not implicitly {@link game.Suggestion.verify|verify} messages.
         * @param message Suggestion message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.Suggestion.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified Suggestion message, length delimited. Does not implicitly {@link game.Suggestion.verify|verify} messages.
         * @param message Suggestion message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.Suggestion.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a Suggestion message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.Suggestion & game.Suggestion.$Shape} Suggestion
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.Suggestion & game.Suggestion.$Shape;

        /**
         * Decodes a Suggestion message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.Suggestion & game.Suggestion.$Shape} Suggestion
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.Suggestion & game.Suggestion.$Shape;

        /**
         * Verifies a Suggestion message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a Suggestion message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns Suggestion
         */
        static fromObject(object: { [k: string]: any }): game.Suggestion;

        /**
         * Creates a plain object from a Suggestion message. Also converts values to other types if specified.
         * @param message Suggestion
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.Suggestion, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this Suggestion to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for Suggestion
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace Suggestion {

        /** Properties of a Suggestion. */
        interface $Properties {

            /** Suggestion id */
            id?: (string|null);

            /** Suggestion playerId */
            playerId?: (string|null);

            /** Suggestion author */
            author?: (string|null);

            /** Suggestion authorPlayer */
            authorPlayer?: (game.PublicPlayer.$Properties|null);

            /** Suggestion text */
            text?: (string|null);

            /** Suggestion at */
            at?: (number|Long|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a Suggestion. */
        type $Shape = game.Suggestion.$Properties;
    }

    /**
     * Properties of a RoomSettings.
     * @deprecated Use game.RoomSettings.$Properties instead.
     */
    interface IRoomSettings extends game.RoomSettings.$Properties {
    }

    /** Represents a RoomSettings. */
    class RoomSettings {

        /**
         * Constructs a new RoomSettings.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.RoomSettings.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** RoomSettings name. */
        name: string;

        /** RoomSettings password. */
        password: string;

        /** RoomSettings gameId. */
        gameId: string;

        /** RoomSettings enablePunishment. */
        enablePunishment: boolean;

        /** RoomSettings punishmentSource. */
        punishmentSource: string;

        /** RoomSettings punishmentId. */
        punishmentId: string;

        /** RoomSettings punishmentIds. */
        punishmentIds: string[];

        /** RoomSettings roomBackgroundImage. */
        roomBackgroundImage: string;

        /** RoomSettings enableTags. */
        enableTags: boolean;

        /** RoomSettings tags. */
        tags: string[];

        /** RoomSettings allowProofImage. */
        allowProofImage: boolean;

        /** RoomSettings tieDoublePunish. */
        tieDoublePunish: boolean;

        /** RoomSettings requireOpponentConfirm. */
        requireOpponentConfirm: boolean;

        /** RoomSettings enableRanked. */
        enableRanked: boolean;

        /** RoomSettings stake. */
        stake: number;

        /** RoomSettings enableRankMultiplier. */
        enableRankMultiplier: boolean;

        /** RoomSettings rankMultiplier. */
        rankMultiplier: number;

        /** RoomSettings enableExtremeRanked. */
        enableExtremeRanked: boolean;

        /** RoomSettings othelloBoardTheme. */
        othelloBoardTheme: string;

        /** RoomSettings tictactoeBoardTheme. */
        tictactoeBoardTheme: string;

        /** RoomSettings liarsDiceMinPlayers. */
        liarsDiceMinPlayers: number;

        /** RoomSettings liarsDiceMaxPlayers. */
        liarsDiceMaxPlayers: number;

        /** RoomSettings gomokuBoardTheme. */
        gomokuBoardTheme: string;

        /** RoomSettings gomokuUndoLimit. */
        gomokuUndoLimit: number;

        /** RoomSettings othelloMoveSeconds. */
        othelloMoveSeconds: number;

        /** RoomSettings othelloGameMinutes. */
        othelloGameMinutes: number;

        /** RoomSettings gomokuMoveSeconds. */
        gomokuMoveSeconds: number;

        /** RoomSettings gomokuGameMinutes. */
        gomokuGameMinutes: number;

        /** RoomSettings jungleBoardTheme. */
        jungleBoardTheme: string;

        /** RoomSettings jungleMoveSeconds. */
        jungleMoveSeconds: number;

        /** RoomSettings jungleGameMinutes. */
        jungleGameMinutes: number;

        /** RoomSettings punishmentTagsIncluded. */
        punishmentTagsIncluded: string[];

        /** RoomSettings punishmentTagsExcluded. */
        punishmentTagsExcluded: string[];

        /** RoomSettings punishmentSeriesId. */
        punishmentSeriesId: string;

        /** RoomSettings chessBoardTheme. */
        chessBoardTheme: string;

        /** RoomSettings chessMoveSeconds. */
        chessMoveSeconds: number;

        /** RoomSettings chessGameMinutes. */
        chessGameMinutes: number;

        /** RoomSettings jungleUndoLimit. */
        jungleUndoLimit: number;

        /** RoomSettings chessUndoLimit. */
        chessUndoLimit: number;

        /** RoomSettings othelloUndoLimit. */
        othelloUndoLimit: number;

        /**
         * Creates a new RoomSettings instance using the specified properties.
         * @param [properties] Properties to set
         * @returns RoomSettings instance
         */
        static create(properties: game.RoomSettings.$Shape): game.RoomSettings & game.RoomSettings.$Shape;
        static create(properties?: game.RoomSettings.$Properties): game.RoomSettings;

        /**
         * Encodes the specified RoomSettings message. Does not implicitly {@link game.RoomSettings.verify|verify} messages.
         * @param message RoomSettings message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.RoomSettings.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified RoomSettings message, length delimited. Does not implicitly {@link game.RoomSettings.verify|verify} messages.
         * @param message RoomSettings message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.RoomSettings.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a RoomSettings message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.RoomSettings & game.RoomSettings.$Shape} RoomSettings
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.RoomSettings & game.RoomSettings.$Shape;

        /**
         * Decodes a RoomSettings message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.RoomSettings & game.RoomSettings.$Shape} RoomSettings
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.RoomSettings & game.RoomSettings.$Shape;

        /**
         * Verifies a RoomSettings message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a RoomSettings message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns RoomSettings
         */
        static fromObject(object: { [k: string]: any }): game.RoomSettings;

        /**
         * Creates a plain object from a RoomSettings message. Also converts values to other types if specified.
         * @param message RoomSettings
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.RoomSettings, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this RoomSettings to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for RoomSettings
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace RoomSettings {

        /** Properties of a RoomSettings. */
        interface $Properties {

            /** RoomSettings name */
            name?: (string|null);

            /** RoomSettings password */
            password?: (string|null);

            /** RoomSettings gameId */
            gameId?: (string|null);

            /** RoomSettings enablePunishment */
            enablePunishment?: (boolean|null);

            /** RoomSettings punishmentSource */
            punishmentSource?: (string|null);

            /** RoomSettings punishmentId */
            punishmentId?: (string|null);

            /** RoomSettings punishmentIds */
            punishmentIds?: (string[]|null);

            /** RoomSettings roomBackgroundImage */
            roomBackgroundImage?: (string|null);

            /** RoomSettings enableTags */
            enableTags?: (boolean|null);

            /** RoomSettings tags */
            tags?: (string[]|null);

            /** RoomSettings allowProofImage */
            allowProofImage?: (boolean|null);

            /** RoomSettings tieDoublePunish */
            tieDoublePunish?: (boolean|null);

            /** RoomSettings requireOpponentConfirm */
            requireOpponentConfirm?: (boolean|null);

            /** RoomSettings enableRanked */
            enableRanked?: (boolean|null);

            /** RoomSettings stake */
            stake?: (number|null);

            /** RoomSettings enableRankMultiplier */
            enableRankMultiplier?: (boolean|null);

            /** RoomSettings rankMultiplier */
            rankMultiplier?: (number|null);

            /** RoomSettings enableExtremeRanked */
            enableExtremeRanked?: (boolean|null);

            /** RoomSettings othelloBoardTheme */
            othelloBoardTheme?: (string|null);

            /** RoomSettings tictactoeBoardTheme */
            tictactoeBoardTheme?: (string|null);

            /** RoomSettings liarsDiceMinPlayers */
            liarsDiceMinPlayers?: (number|null);

            /** RoomSettings liarsDiceMaxPlayers */
            liarsDiceMaxPlayers?: (number|null);

            /** RoomSettings gomokuBoardTheme */
            gomokuBoardTheme?: (string|null);

            /** RoomSettings gomokuUndoLimit */
            gomokuUndoLimit?: (number|null);

            /** RoomSettings othelloMoveSeconds */
            othelloMoveSeconds?: (number|null);

            /** RoomSettings othelloGameMinutes */
            othelloGameMinutes?: (number|null);

            /** RoomSettings gomokuMoveSeconds */
            gomokuMoveSeconds?: (number|null);

            /** RoomSettings gomokuGameMinutes */
            gomokuGameMinutes?: (number|null);

            /** RoomSettings jungleBoardTheme */
            jungleBoardTheme?: (string|null);

            /** RoomSettings jungleMoveSeconds */
            jungleMoveSeconds?: (number|null);

            /** RoomSettings jungleGameMinutes */
            jungleGameMinutes?: (number|null);

            /** RoomSettings punishmentTagsIncluded */
            punishmentTagsIncluded?: (string[]|null);

            /** RoomSettings punishmentTagsExcluded */
            punishmentTagsExcluded?: (string[]|null);

            /** RoomSettings punishmentSeriesId */
            punishmentSeriesId?: (string|null);

            /** RoomSettings chessBoardTheme */
            chessBoardTheme?: (string|null);

            /** RoomSettings chessMoveSeconds */
            chessMoveSeconds?: (number|null);

            /** RoomSettings chessGameMinutes */
            chessGameMinutes?: (number|null);

            /** RoomSettings jungleUndoLimit */
            jungleUndoLimit?: (number|null);

            /** RoomSettings chessUndoLimit */
            chessUndoLimit?: (number|null);

            /** RoomSettings othelloUndoLimit */
            othelloUndoLimit?: (number|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a RoomSettings. */
        type $Shape = game.RoomSettings.$Properties;
    }

    /**
     * Properties of a GomokuUndoRequest.
     * @deprecated Use game.GomokuUndoRequest.$Properties instead.
     */
    interface IGomokuUndoRequest extends game.GomokuUndoRequest.$Properties {
    }

    /** Represents a GomokuUndoRequest. */
    class GomokuUndoRequest {

        /**
         * Constructs a new GomokuUndoRequest.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.GomokuUndoRequest.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** GomokuUndoRequest fromSeat. */
        fromSeat: string;

        /** GomokuUndoRequest toSeat. */
        toSeat: string;

        /** GomokuUndoRequest createdAt. */
        createdAt: (number|Long);

        /** GomokuUndoRequest expiresAt. */
        expiresAt: (number|Long);

        /**
         * Creates a new GomokuUndoRequest instance using the specified properties.
         * @param [properties] Properties to set
         * @returns GomokuUndoRequest instance
         */
        static create(properties: game.GomokuUndoRequest.$Shape): game.GomokuUndoRequest & game.GomokuUndoRequest.$Shape;
        static create(properties?: game.GomokuUndoRequest.$Properties): game.GomokuUndoRequest;

        /**
         * Encodes the specified GomokuUndoRequest message. Does not implicitly {@link game.GomokuUndoRequest.verify|verify} messages.
         * @param message GomokuUndoRequest message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.GomokuUndoRequest.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified GomokuUndoRequest message, length delimited. Does not implicitly {@link game.GomokuUndoRequest.verify|verify} messages.
         * @param message GomokuUndoRequest message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.GomokuUndoRequest.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a GomokuUndoRequest message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.GomokuUndoRequest & game.GomokuUndoRequest.$Shape} GomokuUndoRequest
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.GomokuUndoRequest & game.GomokuUndoRequest.$Shape;

        /**
         * Decodes a GomokuUndoRequest message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.GomokuUndoRequest & game.GomokuUndoRequest.$Shape} GomokuUndoRequest
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.GomokuUndoRequest & game.GomokuUndoRequest.$Shape;

        /**
         * Verifies a GomokuUndoRequest message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a GomokuUndoRequest message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns GomokuUndoRequest
         */
        static fromObject(object: { [k: string]: any }): game.GomokuUndoRequest;

        /**
         * Creates a plain object from a GomokuUndoRequest message. Also converts values to other types if specified.
         * @param message GomokuUndoRequest
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.GomokuUndoRequest, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this GomokuUndoRequest to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for GomokuUndoRequest
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace GomokuUndoRequest {

        /** Properties of a GomokuUndoRequest. */
        interface $Properties {

            /** GomokuUndoRequest fromSeat */
            fromSeat?: (string|null);

            /** GomokuUndoRequest toSeat */
            toSeat?: (string|null);

            /** GomokuUndoRequest createdAt */
            createdAt?: (number|Long|null);

            /** GomokuUndoRequest expiresAt */
            expiresAt?: (number|Long|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a GomokuUndoRequest. */
        type $Shape = game.GomokuUndoRequest.$Properties;
    }

    /**
     * Properties of a GomokuResignRequest.
     * @deprecated Use game.GomokuResignRequest.$Properties instead.
     */
    interface IGomokuResignRequest extends game.GomokuResignRequest.$Properties {
    }

    /** Represents a GomokuResignRequest. */
    class GomokuResignRequest {

        /**
         * Constructs a new GomokuResignRequest.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.GomokuResignRequest.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** GomokuResignRequest fromSeat. */
        fromSeat: string;

        /** GomokuResignRequest toSeat. */
        toSeat: string;

        /** GomokuResignRequest createdAt. */
        createdAt: (number|Long);

        /**
         * Creates a new GomokuResignRequest instance using the specified properties.
         * @param [properties] Properties to set
         * @returns GomokuResignRequest instance
         */
        static create(properties: game.GomokuResignRequest.$Shape): game.GomokuResignRequest & game.GomokuResignRequest.$Shape;
        static create(properties?: game.GomokuResignRequest.$Properties): game.GomokuResignRequest;

        /**
         * Encodes the specified GomokuResignRequest message. Does not implicitly {@link game.GomokuResignRequest.verify|verify} messages.
         * @param message GomokuResignRequest message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.GomokuResignRequest.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified GomokuResignRequest message, length delimited. Does not implicitly {@link game.GomokuResignRequest.verify|verify} messages.
         * @param message GomokuResignRequest message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.GomokuResignRequest.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a GomokuResignRequest message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.GomokuResignRequest & game.GomokuResignRequest.$Shape} GomokuResignRequest
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.GomokuResignRequest & game.GomokuResignRequest.$Shape;

        /**
         * Decodes a GomokuResignRequest message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.GomokuResignRequest & game.GomokuResignRequest.$Shape} GomokuResignRequest
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.GomokuResignRequest & game.GomokuResignRequest.$Shape;

        /**
         * Verifies a GomokuResignRequest message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a GomokuResignRequest message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns GomokuResignRequest
         */
        static fromObject(object: { [k: string]: any }): game.GomokuResignRequest;

        /**
         * Creates a plain object from a GomokuResignRequest message. Also converts values to other types if specified.
         * @param message GomokuResignRequest
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.GomokuResignRequest, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this GomokuResignRequest to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for GomokuResignRequest
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace GomokuResignRequest {

        /** Properties of a GomokuResignRequest. */
        interface $Properties {

            /** GomokuResignRequest fromSeat */
            fromSeat?: (string|null);

            /** GomokuResignRequest toSeat */
            toSeat?: (string|null);

            /** GomokuResignRequest createdAt */
            createdAt?: (number|Long|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a GomokuResignRequest. */
        type $Shape = game.GomokuResignRequest.$Properties;
    }

    /**
     * Properties of a GomokuState.
     * @deprecated Use game.GomokuState.$Properties instead.
     */
    interface IGomokuState extends game.GomokuState.$Properties {
    }

    /** Represents a GomokuState. */
    class GomokuState {

        /**
         * Constructs a new GomokuState.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.GomokuState.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** GomokuState board. */
        board: game.BoardRow.$Properties[];

        /** GomokuState turn. */
        turn: string;

        /** GomokuState blackSeat. */
        blackSeat: string;

        /** GomokuState moveCount. */
        moveCount: number;

        /** GomokuState moves. */
        moves: game.Pos.$Properties[];

        /** GomokuState winningLine. */
        winningLine: game.Pos.$Properties[];

        /** GomokuState rankedDelta. */
        rankedDelta: game.IntPair.$Properties[];

        /** GomokuState undoCount. */
        undoCount: game.IntPair.$Properties[];

        /** GomokuState undoRequest. */
        undoRequest?: (game.GomokuUndoRequest.$Properties|null);

        /** GomokuState resignRequest. */
        resignRequest?: (game.GomokuResignRequest.$Properties|null);

        /** GomokuState ended. */
        ended: boolean;

        /** GomokuState winner. */
        winner: string;

        /** GomokuState moveDeadlineAt. */
        moveDeadlineAt: (number|Long);

        /** GomokuState clockDeadlineAt. */
        clockDeadlineAt: (number|Long);

        /** GomokuState clockRemaining. */
        clockRemaining: game.IntPair.$Properties[];

        /** GomokuState giveawaySeat. */
        giveawaySeat: string;

        /** GomokuState giveawayForcedByMasterName. */
        giveawayForcedByMasterName: string;

        /**
         * Creates a new GomokuState instance using the specified properties.
         * @param [properties] Properties to set
         * @returns GomokuState instance
         */
        static create(properties: game.GomokuState.$Shape): game.GomokuState & game.GomokuState.$Shape;
        static create(properties?: game.GomokuState.$Properties): game.GomokuState;

        /**
         * Encodes the specified GomokuState message. Does not implicitly {@link game.GomokuState.verify|verify} messages.
         * @param message GomokuState message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.GomokuState.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified GomokuState message, length delimited. Does not implicitly {@link game.GomokuState.verify|verify} messages.
         * @param message GomokuState message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.GomokuState.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a GomokuState message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.GomokuState & game.GomokuState.$Shape} GomokuState
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.GomokuState & game.GomokuState.$Shape;

        /**
         * Decodes a GomokuState message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.GomokuState & game.GomokuState.$Shape} GomokuState
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.GomokuState & game.GomokuState.$Shape;

        /**
         * Verifies a GomokuState message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a GomokuState message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns GomokuState
         */
        static fromObject(object: { [k: string]: any }): game.GomokuState;

        /**
         * Creates a plain object from a GomokuState message. Also converts values to other types if specified.
         * @param message GomokuState
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.GomokuState, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this GomokuState to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for GomokuState
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace GomokuState {

        /** Properties of a GomokuState. */
        interface $Properties {

            /** GomokuState board */
            board?: (game.BoardRow.$Properties[]|null);

            /** GomokuState turn */
            turn?: (string|null);

            /** GomokuState blackSeat */
            blackSeat?: (string|null);

            /** GomokuState moveCount */
            moveCount?: (number|null);

            /** GomokuState moves */
            moves?: (game.Pos.$Properties[]|null);

            /** GomokuState winningLine */
            winningLine?: (game.Pos.$Properties[]|null);

            /** GomokuState rankedDelta */
            rankedDelta?: (game.IntPair.$Properties[]|null);

            /** GomokuState undoCount */
            undoCount?: (game.IntPair.$Properties[]|null);

            /** GomokuState undoRequest */
            undoRequest?: (game.GomokuUndoRequest.$Properties|null);

            /** GomokuState resignRequest */
            resignRequest?: (game.GomokuResignRequest.$Properties|null);

            /** GomokuState ended */
            ended?: (boolean|null);

            /** GomokuState winner */
            winner?: (string|null);

            /** GomokuState moveDeadlineAt */
            moveDeadlineAt?: (number|Long|null);

            /** GomokuState clockDeadlineAt */
            clockDeadlineAt?: (number|Long|null);

            /** GomokuState clockRemaining */
            clockRemaining?: (game.IntPair.$Properties[]|null);

            /** GomokuState giveawaySeat */
            giveawaySeat?: (string|null);

            /** GomokuState giveawayForcedByMasterName */
            giveawayForcedByMasterName?: (string|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a GomokuState. */
        type $Shape = game.GomokuState.$Properties;
    }

    /**
     * Properties of a JungleResignRequest.
     * @deprecated Use game.JungleResignRequest.$Properties instead.
     */
    interface IJungleResignRequest extends game.JungleResignRequest.$Properties {
    }

    /** Represents a JungleResignRequest. */
    class JungleResignRequest {

        /**
         * Constructs a new JungleResignRequest.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.JungleResignRequest.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** JungleResignRequest fromSeat. */
        fromSeat: string;

        /** JungleResignRequest toSeat. */
        toSeat: string;

        /** JungleResignRequest createdAt. */
        createdAt: (number|Long);

        /**
         * Creates a new JungleResignRequest instance using the specified properties.
         * @param [properties] Properties to set
         * @returns JungleResignRequest instance
         */
        static create(properties: game.JungleResignRequest.$Shape): game.JungleResignRequest & game.JungleResignRequest.$Shape;
        static create(properties?: game.JungleResignRequest.$Properties): game.JungleResignRequest;

        /**
         * Encodes the specified JungleResignRequest message. Does not implicitly {@link game.JungleResignRequest.verify|verify} messages.
         * @param message JungleResignRequest message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.JungleResignRequest.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified JungleResignRequest message, length delimited. Does not implicitly {@link game.JungleResignRequest.verify|verify} messages.
         * @param message JungleResignRequest message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.JungleResignRequest.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a JungleResignRequest message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.JungleResignRequest & game.JungleResignRequest.$Shape} JungleResignRequest
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.JungleResignRequest & game.JungleResignRequest.$Shape;

        /**
         * Decodes a JungleResignRequest message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.JungleResignRequest & game.JungleResignRequest.$Shape} JungleResignRequest
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.JungleResignRequest & game.JungleResignRequest.$Shape;

        /**
         * Verifies a JungleResignRequest message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a JungleResignRequest message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns JungleResignRequest
         */
        static fromObject(object: { [k: string]: any }): game.JungleResignRequest;

        /**
         * Creates a plain object from a JungleResignRequest message. Also converts values to other types if specified.
         * @param message JungleResignRequest
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.JungleResignRequest, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this JungleResignRequest to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for JungleResignRequest
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace JungleResignRequest {

        /** Properties of a JungleResignRequest. */
        interface $Properties {

            /** JungleResignRequest fromSeat */
            fromSeat?: (string|null);

            /** JungleResignRequest toSeat */
            toSeat?: (string|null);

            /** JungleResignRequest createdAt */
            createdAt?: (number|Long|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a JungleResignRequest. */
        type $Shape = game.JungleResignRequest.$Properties;
    }

    /**
     * Properties of a JungleUndoRequest.
     * @deprecated Use game.JungleUndoRequest.$Properties instead.
     */
    interface IJungleUndoRequest extends game.JungleUndoRequest.$Properties {
    }

    /** Represents a JungleUndoRequest. */
    class JungleUndoRequest {

        /**
         * Constructs a new JungleUndoRequest.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.JungleUndoRequest.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** JungleUndoRequest fromSeat. */
        fromSeat: string;

        /** JungleUndoRequest toSeat. */
        toSeat: string;

        /** JungleUndoRequest createdAt. */
        createdAt: (number|Long);

        /** JungleUndoRequest expiresAt. */
        expiresAt: (number|Long);

        /**
         * Creates a new JungleUndoRequest instance using the specified properties.
         * @param [properties] Properties to set
         * @returns JungleUndoRequest instance
         */
        static create(properties: game.JungleUndoRequest.$Shape): game.JungleUndoRequest & game.JungleUndoRequest.$Shape;
        static create(properties?: game.JungleUndoRequest.$Properties): game.JungleUndoRequest;

        /**
         * Encodes the specified JungleUndoRequest message. Does not implicitly {@link game.JungleUndoRequest.verify|verify} messages.
         * @param message JungleUndoRequest message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.JungleUndoRequest.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified JungleUndoRequest message, length delimited. Does not implicitly {@link game.JungleUndoRequest.verify|verify} messages.
         * @param message JungleUndoRequest message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.JungleUndoRequest.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a JungleUndoRequest message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.JungleUndoRequest & game.JungleUndoRequest.$Shape} JungleUndoRequest
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.JungleUndoRequest & game.JungleUndoRequest.$Shape;

        /**
         * Decodes a JungleUndoRequest message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.JungleUndoRequest & game.JungleUndoRequest.$Shape} JungleUndoRequest
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.JungleUndoRequest & game.JungleUndoRequest.$Shape;

        /**
         * Verifies a JungleUndoRequest message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a JungleUndoRequest message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns JungleUndoRequest
         */
        static fromObject(object: { [k: string]: any }): game.JungleUndoRequest;

        /**
         * Creates a plain object from a JungleUndoRequest message. Also converts values to other types if specified.
         * @param message JungleUndoRequest
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.JungleUndoRequest, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this JungleUndoRequest to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for JungleUndoRequest
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace JungleUndoRequest {

        /** Properties of a JungleUndoRequest. */
        interface $Properties {

            /** JungleUndoRequest fromSeat */
            fromSeat?: (string|null);

            /** JungleUndoRequest toSeat */
            toSeat?: (string|null);

            /** JungleUndoRequest createdAt */
            createdAt?: (number|Long|null);

            /** JungleUndoRequest expiresAt */
            expiresAt?: (number|Long|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a JungleUndoRequest. */
        type $Shape = game.JungleUndoRequest.$Properties;
    }

    /**
     * Properties of a JungleState.
     * @deprecated Use game.JungleState.$Properties instead.
     */
    interface IJungleState extends game.JungleState.$Properties {
    }

    /** Represents a JungleState. */
    class JungleState {

        /**
         * Constructs a new JungleState.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.JungleState.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** JungleState board. */
        board: game.BoardRow.$Properties[];

        /** JungleState turn. */
        turn: string;

        /** JungleState moveCount. */
        moveCount: number;

        /** JungleState lastFrom. */
        lastFrom?: (game.Pos.$Properties|null);

        /** JungleState lastTo. */
        lastTo?: (game.Pos.$Properties|null);

        /** JungleState rankedDelta. */
        rankedDelta: game.IntPair.$Properties[];

        /** JungleState resignRequest. */
        resignRequest?: (game.JungleResignRequest.$Properties|null);

        /** JungleState ended. */
        ended: boolean;

        /** JungleState winner. */
        winner: string;

        /** JungleState moveDeadlineAt. */
        moveDeadlineAt: (number|Long);

        /** JungleState clockDeadlineAt. */
        clockDeadlineAt: (number|Long);

        /** JungleState clockRemaining. */
        clockRemaining: game.IntPair.$Properties[];

        /** JungleState undoCount. */
        undoCount: game.IntPair.$Properties[];

        /** JungleState undoRequest. */
        undoRequest?: (game.JungleUndoRequest.$Properties|null);

        /**
         * Creates a new JungleState instance using the specified properties.
         * @param [properties] Properties to set
         * @returns JungleState instance
         */
        static create(properties: game.JungleState.$Shape): game.JungleState & game.JungleState.$Shape;
        static create(properties?: game.JungleState.$Properties): game.JungleState;

        /**
         * Encodes the specified JungleState message. Does not implicitly {@link game.JungleState.verify|verify} messages.
         * @param message JungleState message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.JungleState.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified JungleState message, length delimited. Does not implicitly {@link game.JungleState.verify|verify} messages.
         * @param message JungleState message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.JungleState.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a JungleState message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.JungleState & game.JungleState.$Shape} JungleState
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.JungleState & game.JungleState.$Shape;

        /**
         * Decodes a JungleState message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.JungleState & game.JungleState.$Shape} JungleState
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.JungleState & game.JungleState.$Shape;

        /**
         * Verifies a JungleState message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a JungleState message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns JungleState
         */
        static fromObject(object: { [k: string]: any }): game.JungleState;

        /**
         * Creates a plain object from a JungleState message. Also converts values to other types if specified.
         * @param message JungleState
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.JungleState, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this JungleState to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for JungleState
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace JungleState {

        /** Properties of a JungleState. */
        interface $Properties {

            /** JungleState board */
            board?: (game.BoardRow.$Properties[]|null);

            /** JungleState turn */
            turn?: (string|null);

            /** JungleState moveCount */
            moveCount?: (number|null);

            /** JungleState lastFrom */
            lastFrom?: (game.Pos.$Properties|null);

            /** JungleState lastTo */
            lastTo?: (game.Pos.$Properties|null);

            /** JungleState rankedDelta */
            rankedDelta?: (game.IntPair.$Properties[]|null);

            /** JungleState resignRequest */
            resignRequest?: (game.JungleResignRequest.$Properties|null);

            /** JungleState ended */
            ended?: (boolean|null);

            /** JungleState winner */
            winner?: (string|null);

            /** JungleState moveDeadlineAt */
            moveDeadlineAt?: (number|Long|null);

            /** JungleState clockDeadlineAt */
            clockDeadlineAt?: (number|Long|null);

            /** JungleState clockRemaining */
            clockRemaining?: (game.IntPair.$Properties[]|null);

            /** JungleState undoCount */
            undoCount?: (game.IntPair.$Properties[]|null);

            /** JungleState undoRequest */
            undoRequest?: (game.JungleUndoRequest.$Properties|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a JungleState. */
        type $Shape = game.JungleState.$Properties;
    }

    /**
     * Properties of a ChessResignRequest.
     * @deprecated Use game.ChessResignRequest.$Properties instead.
     */
    interface IChessResignRequest extends game.ChessResignRequest.$Properties {
    }

    /** Represents a ChessResignRequest. */
    class ChessResignRequest {

        /**
         * Constructs a new ChessResignRequest.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.ChessResignRequest.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** ChessResignRequest fromSeat. */
        fromSeat: string;

        /** ChessResignRequest toSeat. */
        toSeat: string;

        /** ChessResignRequest createdAt. */
        createdAt: (number|Long);

        /**
         * Creates a new ChessResignRequest instance using the specified properties.
         * @param [properties] Properties to set
         * @returns ChessResignRequest instance
         */
        static create(properties: game.ChessResignRequest.$Shape): game.ChessResignRequest & game.ChessResignRequest.$Shape;
        static create(properties?: game.ChessResignRequest.$Properties): game.ChessResignRequest;

        /**
         * Encodes the specified ChessResignRequest message. Does not implicitly {@link game.ChessResignRequest.verify|verify} messages.
         * @param message ChessResignRequest message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.ChessResignRequest.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified ChessResignRequest message, length delimited. Does not implicitly {@link game.ChessResignRequest.verify|verify} messages.
         * @param message ChessResignRequest message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.ChessResignRequest.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a ChessResignRequest message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.ChessResignRequest & game.ChessResignRequest.$Shape} ChessResignRequest
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.ChessResignRequest & game.ChessResignRequest.$Shape;

        /**
         * Decodes a ChessResignRequest message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.ChessResignRequest & game.ChessResignRequest.$Shape} ChessResignRequest
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.ChessResignRequest & game.ChessResignRequest.$Shape;

        /**
         * Verifies a ChessResignRequest message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a ChessResignRequest message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns ChessResignRequest
         */
        static fromObject(object: { [k: string]: any }): game.ChessResignRequest;

        /**
         * Creates a plain object from a ChessResignRequest message. Also converts values to other types if specified.
         * @param message ChessResignRequest
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.ChessResignRequest, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this ChessResignRequest to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for ChessResignRequest
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace ChessResignRequest {

        /** Properties of a ChessResignRequest. */
        interface $Properties {

            /** ChessResignRequest fromSeat */
            fromSeat?: (string|null);

            /** ChessResignRequest toSeat */
            toSeat?: (string|null);

            /** ChessResignRequest createdAt */
            createdAt?: (number|Long|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a ChessResignRequest. */
        type $Shape = game.ChessResignRequest.$Properties;
    }

    /**
     * Properties of a ChessUndoRequest.
     * @deprecated Use game.ChessUndoRequest.$Properties instead.
     */
    interface IChessUndoRequest extends game.ChessUndoRequest.$Properties {
    }

    /** Represents a ChessUndoRequest. */
    class ChessUndoRequest {

        /**
         * Constructs a new ChessUndoRequest.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.ChessUndoRequest.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** ChessUndoRequest fromSeat. */
        fromSeat: string;

        /** ChessUndoRequest toSeat. */
        toSeat: string;

        /** ChessUndoRequest createdAt. */
        createdAt: (number|Long);

        /** ChessUndoRequest expiresAt. */
        expiresAt: (number|Long);

        /**
         * Creates a new ChessUndoRequest instance using the specified properties.
         * @param [properties] Properties to set
         * @returns ChessUndoRequest instance
         */
        static create(properties: game.ChessUndoRequest.$Shape): game.ChessUndoRequest & game.ChessUndoRequest.$Shape;
        static create(properties?: game.ChessUndoRequest.$Properties): game.ChessUndoRequest;

        /**
         * Encodes the specified ChessUndoRequest message. Does not implicitly {@link game.ChessUndoRequest.verify|verify} messages.
         * @param message ChessUndoRequest message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.ChessUndoRequest.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified ChessUndoRequest message, length delimited. Does not implicitly {@link game.ChessUndoRequest.verify|verify} messages.
         * @param message ChessUndoRequest message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.ChessUndoRequest.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a ChessUndoRequest message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.ChessUndoRequest & game.ChessUndoRequest.$Shape} ChessUndoRequest
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.ChessUndoRequest & game.ChessUndoRequest.$Shape;

        /**
         * Decodes a ChessUndoRequest message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.ChessUndoRequest & game.ChessUndoRequest.$Shape} ChessUndoRequest
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.ChessUndoRequest & game.ChessUndoRequest.$Shape;

        /**
         * Verifies a ChessUndoRequest message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a ChessUndoRequest message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns ChessUndoRequest
         */
        static fromObject(object: { [k: string]: any }): game.ChessUndoRequest;

        /**
         * Creates a plain object from a ChessUndoRequest message. Also converts values to other types if specified.
         * @param message ChessUndoRequest
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.ChessUndoRequest, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this ChessUndoRequest to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for ChessUndoRequest
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace ChessUndoRequest {

        /** Properties of a ChessUndoRequest. */
        interface $Properties {

            /** ChessUndoRequest fromSeat */
            fromSeat?: (string|null);

            /** ChessUndoRequest toSeat */
            toSeat?: (string|null);

            /** ChessUndoRequest createdAt */
            createdAt?: (number|Long|null);

            /** ChessUndoRequest expiresAt */
            expiresAt?: (number|Long|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a ChessUndoRequest. */
        type $Shape = game.ChessUndoRequest.$Properties;
    }

    /**
     * Properties of a ChessMove.
     * @deprecated Use game.ChessMove.$Properties instead.
     */
    interface IChessMove extends game.ChessMove.$Properties {
    }

    /** Represents a ChessMove. */
    class ChessMove {

        /**
         * Constructs a new ChessMove.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.ChessMove.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** ChessMove from. */
        from?: (game.Pos.$Properties|null);

        /** ChessMove to. */
        to?: (game.Pos.$Properties|null);

        /** ChessMove promote. */
        promote: string;

        /**
         * Creates a new ChessMove instance using the specified properties.
         * @param [properties] Properties to set
         * @returns ChessMove instance
         */
        static create(properties: game.ChessMove.$Shape): game.ChessMove & game.ChessMove.$Shape;
        static create(properties?: game.ChessMove.$Properties): game.ChessMove;

        /**
         * Encodes the specified ChessMove message. Does not implicitly {@link game.ChessMove.verify|verify} messages.
         * @param message ChessMove message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.ChessMove.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified ChessMove message, length delimited. Does not implicitly {@link game.ChessMove.verify|verify} messages.
         * @param message ChessMove message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.ChessMove.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a ChessMove message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.ChessMove & game.ChessMove.$Shape} ChessMove
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.ChessMove & game.ChessMove.$Shape;

        /**
         * Decodes a ChessMove message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.ChessMove & game.ChessMove.$Shape} ChessMove
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.ChessMove & game.ChessMove.$Shape;

        /**
         * Verifies a ChessMove message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a ChessMove message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns ChessMove
         */
        static fromObject(object: { [k: string]: any }): game.ChessMove;

        /**
         * Creates a plain object from a ChessMove message. Also converts values to other types if specified.
         * @param message ChessMove
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.ChessMove, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this ChessMove to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for ChessMove
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace ChessMove {

        /** Properties of a ChessMove. */
        interface $Properties {

            /** ChessMove from */
            from?: (game.Pos.$Properties|null);

            /** ChessMove to */
            to?: (game.Pos.$Properties|null);

            /** ChessMove promote */
            promote?: (string|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a ChessMove. */
        type $Shape = game.ChessMove.$Properties;
    }

    /**
     * Properties of a ChessState.
     * @deprecated Use game.ChessState.$Properties instead.
     */
    interface IChessState extends game.ChessState.$Properties {
    }

    /** Represents a ChessState. */
    class ChessState {

        /**
         * Constructs a new ChessState.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.ChessState.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** ChessState board. */
        board: game.BoardRow.$Properties[];

        /** ChessState turn. */
        turn: string;

        /** ChessState whiteSeat. */
        whiteSeat: string;

        /** ChessState moveCount. */
        moveCount: number;

        /** ChessState lastFrom. */
        lastFrom?: (game.Pos.$Properties|null);

        /** ChessState lastTo. */
        lastTo?: (game.Pos.$Properties|null);

        /** ChessState rankedDelta. */
        rankedDelta: game.IntPair.$Properties[];

        /** ChessState resignRequest. */
        resignRequest?: (game.ChessResignRequest.$Properties|null);

        /** ChessState ended. */
        ended: boolean;

        /** ChessState winner. */
        winner: string;

        /** ChessState moveDeadlineAt. */
        moveDeadlineAt: (number|Long);

        /** ChessState clockDeadlineAt. */
        clockDeadlineAt: (number|Long);

        /** ChessState clockRemaining. */
        clockRemaining: game.IntPair.$Properties[];

        /** ChessState castlingWhiteK. */
        castlingWhiteK: boolean;

        /** ChessState castlingWhiteQ. */
        castlingWhiteQ: boolean;

        /** ChessState castlingBlackK. */
        castlingBlackK: boolean;

        /** ChessState castlingBlackQ. */
        castlingBlackQ: boolean;

        /** ChessState enPassant. */
        enPassant?: (game.Pos.$Properties|null);

        /** ChessState halfmoveClock. */
        halfmoveClock: number;

        /** ChessState inCheck. */
        inCheck: boolean;

        /** ChessState legalMoves. */
        legalMoves: game.ChessMove.$Properties[];

        /** ChessState undoCount. */
        undoCount: game.IntPair.$Properties[];

        /** ChessState undoRequest. */
        undoRequest?: (game.ChessUndoRequest.$Properties|null);

        /**
         * Creates a new ChessState instance using the specified properties.
         * @param [properties] Properties to set
         * @returns ChessState instance
         */
        static create(properties: game.ChessState.$Shape): game.ChessState & game.ChessState.$Shape;
        static create(properties?: game.ChessState.$Properties): game.ChessState;

        /**
         * Encodes the specified ChessState message. Does not implicitly {@link game.ChessState.verify|verify} messages.
         * @param message ChessState message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.ChessState.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified ChessState message, length delimited. Does not implicitly {@link game.ChessState.verify|verify} messages.
         * @param message ChessState message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.ChessState.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a ChessState message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.ChessState & game.ChessState.$Shape} ChessState
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.ChessState & game.ChessState.$Shape;

        /**
         * Decodes a ChessState message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.ChessState & game.ChessState.$Shape} ChessState
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.ChessState & game.ChessState.$Shape;

        /**
         * Verifies a ChessState message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a ChessState message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns ChessState
         */
        static fromObject(object: { [k: string]: any }): game.ChessState;

        /**
         * Creates a plain object from a ChessState message. Also converts values to other types if specified.
         * @param message ChessState
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.ChessState, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this ChessState to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for ChessState
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace ChessState {

        /** Properties of a ChessState. */
        interface $Properties {

            /** ChessState board */
            board?: (game.BoardRow.$Properties[]|null);

            /** ChessState turn */
            turn?: (string|null);

            /** ChessState whiteSeat */
            whiteSeat?: (string|null);

            /** ChessState moveCount */
            moveCount?: (number|null);

            /** ChessState lastFrom */
            lastFrom?: (game.Pos.$Properties|null);

            /** ChessState lastTo */
            lastTo?: (game.Pos.$Properties|null);

            /** ChessState rankedDelta */
            rankedDelta?: (game.IntPair.$Properties[]|null);

            /** ChessState resignRequest */
            resignRequest?: (game.ChessResignRequest.$Properties|null);

            /** ChessState ended */
            ended?: (boolean|null);

            /** ChessState winner */
            winner?: (string|null);

            /** ChessState moveDeadlineAt */
            moveDeadlineAt?: (number|Long|null);

            /** ChessState clockDeadlineAt */
            clockDeadlineAt?: (number|Long|null);

            /** ChessState clockRemaining */
            clockRemaining?: (game.IntPair.$Properties[]|null);

            /** ChessState castlingWhiteK */
            castlingWhiteK?: (boolean|null);

            /** ChessState castlingWhiteQ */
            castlingWhiteQ?: (boolean|null);

            /** ChessState castlingBlackK */
            castlingBlackK?: (boolean|null);

            /** ChessState castlingBlackQ */
            castlingBlackQ?: (boolean|null);

            /** ChessState enPassant */
            enPassant?: (game.Pos.$Properties|null);

            /** ChessState halfmoveClock */
            halfmoveClock?: (number|null);

            /** ChessState inCheck */
            inCheck?: (boolean|null);

            /** ChessState legalMoves */
            legalMoves?: (game.ChessMove.$Properties[]|null);

            /** ChessState undoCount */
            undoCount?: (game.IntPair.$Properties[]|null);

            /** ChessState undoRequest */
            undoRequest?: (game.ChessUndoRequest.$Properties|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a ChessState. */
        type $Shape = game.ChessState.$Properties;
    }

    /**
     * Properties of a PunishmentProof.
     * @deprecated Use game.PunishmentProof.$Properties instead.
     */
    interface IPunishmentProof extends game.PunishmentProof.$Properties {
    }

    /** Represents a PunishmentProof. */
    class PunishmentProof {

        /**
         * Constructs a new PunishmentProof.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.PunishmentProof.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** PunishmentProof playerId. */
        playerId: string;

        /** PunishmentProof text. */
        text: string;

        /** PunishmentProof imageUrl. */
        imageUrl: string;

        /** PunishmentProof taskText. */
        taskText: string;

        /** PunishmentProof status. */
        status: string;

        /** PunishmentProof confirmedBy. */
        confirmedBy: string;

        /** PunishmentProof reviewedBy. */
        reviewedBy: string;

        /** PunishmentProof reviewedAt. */
        reviewedAt: (number|Long);

        /** PunishmentProof rejectReason. */
        rejectReason: string;

        /** PunishmentProof redoTaskText. */
        redoTaskText: string;

        /** PunishmentProof submittedAt. */
        submittedAt: (number|Long);

        /**
         * Creates a new PunishmentProof instance using the specified properties.
         * @param [properties] Properties to set
         * @returns PunishmentProof instance
         */
        static create(properties: game.PunishmentProof.$Shape): game.PunishmentProof & game.PunishmentProof.$Shape;
        static create(properties?: game.PunishmentProof.$Properties): game.PunishmentProof;

        /**
         * Encodes the specified PunishmentProof message. Does not implicitly {@link game.PunishmentProof.verify|verify} messages.
         * @param message PunishmentProof message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.PunishmentProof.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified PunishmentProof message, length delimited. Does not implicitly {@link game.PunishmentProof.verify|verify} messages.
         * @param message PunishmentProof message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.PunishmentProof.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a PunishmentProof message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.PunishmentProof & game.PunishmentProof.$Shape} PunishmentProof
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.PunishmentProof & game.PunishmentProof.$Shape;

        /**
         * Decodes a PunishmentProof message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.PunishmentProof & game.PunishmentProof.$Shape} PunishmentProof
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.PunishmentProof & game.PunishmentProof.$Shape;

        /**
         * Verifies a PunishmentProof message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a PunishmentProof message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns PunishmentProof
         */
        static fromObject(object: { [k: string]: any }): game.PunishmentProof;

        /**
         * Creates a plain object from a PunishmentProof message. Also converts values to other types if specified.
         * @param message PunishmentProof
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.PunishmentProof, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this PunishmentProof to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for PunishmentProof
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace PunishmentProof {

        /** Properties of a PunishmentProof. */
        interface $Properties {

            /** PunishmentProof playerId */
            playerId?: (string|null);

            /** PunishmentProof text */
            text?: (string|null);

            /** PunishmentProof imageUrl */
            imageUrl?: (string|null);

            /** PunishmentProof taskText */
            taskText?: (string|null);

            /** PunishmentProof status */
            status?: (string|null);

            /** PunishmentProof confirmedBy */
            confirmedBy?: (string|null);

            /** PunishmentProof reviewedBy */
            reviewedBy?: (string|null);

            /** PunishmentProof reviewedAt */
            reviewedAt?: (number|Long|null);

            /** PunishmentProof rejectReason */
            rejectReason?: (string|null);

            /** PunishmentProof redoTaskText */
            redoTaskText?: (string|null);

            /** PunishmentProof submittedAt */
            submittedAt?: (number|Long|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a PunishmentProof. */
        type $Shape = game.PunishmentProof.$Properties;
    }

    /**
     * Properties of a SeatStats.
     * @deprecated Use game.SeatStats.$Properties instead.
     */
    interface ISeatStats extends game.SeatStats.$Properties {
    }

    /** Represents a SeatStats. */
    class SeatStats {

        /**
         * Constructs a new SeatStats.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.SeatStats.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** SeatStats wins. */
        wins: number;

        /** SeatStats losses. */
        losses: number;

        /** SeatStats draws. */
        draws: number;

        /** SeatStats punishments. */
        punishments: number;

        /**
         * Creates a new SeatStats instance using the specified properties.
         * @param [properties] Properties to set
         * @returns SeatStats instance
         */
        static create(properties: game.SeatStats.$Shape): game.SeatStats & game.SeatStats.$Shape;
        static create(properties?: game.SeatStats.$Properties): game.SeatStats;

        /**
         * Encodes the specified SeatStats message. Does not implicitly {@link game.SeatStats.verify|verify} messages.
         * @param message SeatStats message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.SeatStats.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified SeatStats message, length delimited. Does not implicitly {@link game.SeatStats.verify|verify} messages.
         * @param message SeatStats message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.SeatStats.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a SeatStats message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.SeatStats & game.SeatStats.$Shape} SeatStats
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.SeatStats & game.SeatStats.$Shape;

        /**
         * Decodes a SeatStats message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.SeatStats & game.SeatStats.$Shape} SeatStats
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.SeatStats & game.SeatStats.$Shape;

        /**
         * Verifies a SeatStats message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a SeatStats message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns SeatStats
         */
        static fromObject(object: { [k: string]: any }): game.SeatStats;

        /**
         * Creates a plain object from a SeatStats message. Also converts values to other types if specified.
         * @param message SeatStats
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.SeatStats, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this SeatStats to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for SeatStats
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace SeatStats {

        /** Properties of a SeatStats. */
        interface $Properties {

            /** SeatStats wins */
            wins?: (number|null);

            /** SeatStats losses */
            losses?: (number|null);

            /** SeatStats draws */
            draws?: (number|null);

            /** SeatStats punishments */
            punishments?: (number|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a SeatStats. */
        type $Shape = game.SeatStats.$Properties;
    }

    /**
     * Properties of an OthelloScore.
     * @deprecated Use game.OthelloScore.$Properties instead.
     */
    interface IOthelloScore extends game.OthelloScore.$Properties {
    }

    /** Represents an OthelloScore. */
    class OthelloScore {

        /**
         * Constructs a new OthelloScore.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.OthelloScore.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** OthelloScore black. */
        black: number;

        /** OthelloScore white. */
        white: number;

        /**
         * Creates a new OthelloScore instance using the specified properties.
         * @param [properties] Properties to set
         * @returns OthelloScore instance
         */
        static create(properties: game.OthelloScore.$Shape): game.OthelloScore & game.OthelloScore.$Shape;
        static create(properties?: game.OthelloScore.$Properties): game.OthelloScore;

        /**
         * Encodes the specified OthelloScore message. Does not implicitly {@link game.OthelloScore.verify|verify} messages.
         * @param message OthelloScore message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.OthelloScore.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified OthelloScore message, length delimited. Does not implicitly {@link game.OthelloScore.verify|verify} messages.
         * @param message OthelloScore message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.OthelloScore.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes an OthelloScore message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.OthelloScore & game.OthelloScore.$Shape} OthelloScore
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.OthelloScore & game.OthelloScore.$Shape;

        /**
         * Decodes an OthelloScore message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.OthelloScore & game.OthelloScore.$Shape} OthelloScore
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.OthelloScore & game.OthelloScore.$Shape;

        /**
         * Verifies an OthelloScore message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates an OthelloScore message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns OthelloScore
         */
        static fromObject(object: { [k: string]: any }): game.OthelloScore;

        /**
         * Creates a plain object from an OthelloScore message. Also converts values to other types if specified.
         * @param message OthelloScore
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.OthelloScore, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this OthelloScore to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for OthelloScore
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace OthelloScore {

        /** Properties of an OthelloScore. */
        interface $Properties {

            /** OthelloScore black */
            black?: (number|null);

            /** OthelloScore white */
            white?: (number|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of an OthelloScore. */
        type $Shape = game.OthelloScore.$Properties;
    }

    /**
     * Properties of a PunishmentTask.
     * @deprecated Use game.PunishmentTask.$Properties instead.
     */
    interface IPunishmentTask extends game.PunishmentTask.$Properties {
    }

    /** Represents a PunishmentTask. */
    class PunishmentTask {

        /**
         * Constructs a new PunishmentTask.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.PunishmentTask.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** PunishmentTask playerId. */
        playerId: string;

        /** PunishmentTask playerName. */
        playerName: string;

        /** PunishmentTask factionId. */
        factionId: string;

        /** PunishmentTask factionLabel. */
        factionLabel: string;

        /** PunishmentTask taskText. */
        taskText: string;

        /** PunishmentTask backgroundImage. */
        backgroundImage: string;

        /** PunishmentTask backgroundOpacity. */
        backgroundOpacity: number;

        /** PunishmentTask assignedBy. */
        assignedBy: string;

        /** PunishmentTask assignedByName. */
        assignedByName: string;

        /** PunishmentTask typeName. */
        typeName: string;

        /**
         * Creates a new PunishmentTask instance using the specified properties.
         * @param [properties] Properties to set
         * @returns PunishmentTask instance
         */
        static create(properties: game.PunishmentTask.$Shape): game.PunishmentTask & game.PunishmentTask.$Shape;
        static create(properties?: game.PunishmentTask.$Properties): game.PunishmentTask;

        /**
         * Encodes the specified PunishmentTask message. Does not implicitly {@link game.PunishmentTask.verify|verify} messages.
         * @param message PunishmentTask message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.PunishmentTask.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified PunishmentTask message, length delimited. Does not implicitly {@link game.PunishmentTask.verify|verify} messages.
         * @param message PunishmentTask message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.PunishmentTask.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a PunishmentTask message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.PunishmentTask & game.PunishmentTask.$Shape} PunishmentTask
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.PunishmentTask & game.PunishmentTask.$Shape;

        /**
         * Decodes a PunishmentTask message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.PunishmentTask & game.PunishmentTask.$Shape} PunishmentTask
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.PunishmentTask & game.PunishmentTask.$Shape;

        /**
         * Verifies a PunishmentTask message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a PunishmentTask message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns PunishmentTask
         */
        static fromObject(object: { [k: string]: any }): game.PunishmentTask;

        /**
         * Creates a plain object from a PunishmentTask message. Also converts values to other types if specified.
         * @param message PunishmentTask
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.PunishmentTask, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this PunishmentTask to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for PunishmentTask
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace PunishmentTask {

        /** Properties of a PunishmentTask. */
        interface $Properties {

            /** PunishmentTask playerId */
            playerId?: (string|null);

            /** PunishmentTask playerName */
            playerName?: (string|null);

            /** PunishmentTask factionId */
            factionId?: (string|null);

            /** PunishmentTask factionLabel */
            factionLabel?: (string|null);

            /** PunishmentTask taskText */
            taskText?: (string|null);

            /** PunishmentTask backgroundImage */
            backgroundImage?: (string|null);

            /** PunishmentTask backgroundOpacity */
            backgroundOpacity?: (number|null);

            /** PunishmentTask assignedBy */
            assignedBy?: (string|null);

            /** PunishmentTask assignedByName */
            assignedByName?: (string|null);

            /** PunishmentTask typeName */
            typeName?: (string|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a PunishmentTask. */
        type $Shape = game.PunishmentTask.$Properties;
    }

    /**
     * Properties of a HistoryProof.
     * @deprecated Use game.HistoryProof.$Properties instead.
     */
    interface IHistoryProof extends game.HistoryProof.$Properties {
    }

    /** Represents a HistoryProof. */
    class HistoryProof {

        /**
         * Constructs a new HistoryProof.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.HistoryProof.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** HistoryProof playerId. */
        playerId: string;

        /** HistoryProof playerName. */
        playerName: string;

        /** HistoryProof text. */
        text: string;

        /** HistoryProof imageUrl. */
        imageUrl: string;

        /** HistoryProof taskText. */
        taskText: string;

        /** HistoryProof status. */
        status: string;

        /** HistoryProof reviewedBy. */
        reviewedBy: string;

        /** HistoryProof reviewedAt. */
        reviewedAt: (number|Long);

        /** HistoryProof rejectReason. */
        rejectReason: string;

        /** HistoryProof redoTaskText. */
        redoTaskText: string;

        /** HistoryProof submittedAt. */
        submittedAt: (number|Long);

        /**
         * Creates a new HistoryProof instance using the specified properties.
         * @param [properties] Properties to set
         * @returns HistoryProof instance
         */
        static create(properties: game.HistoryProof.$Shape): game.HistoryProof & game.HistoryProof.$Shape;
        static create(properties?: game.HistoryProof.$Properties): game.HistoryProof;

        /**
         * Encodes the specified HistoryProof message. Does not implicitly {@link game.HistoryProof.verify|verify} messages.
         * @param message HistoryProof message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.HistoryProof.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified HistoryProof message, length delimited. Does not implicitly {@link game.HistoryProof.verify|verify} messages.
         * @param message HistoryProof message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.HistoryProof.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a HistoryProof message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.HistoryProof & game.HistoryProof.$Shape} HistoryProof
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.HistoryProof & game.HistoryProof.$Shape;

        /**
         * Decodes a HistoryProof message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.HistoryProof & game.HistoryProof.$Shape} HistoryProof
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.HistoryProof & game.HistoryProof.$Shape;

        /**
         * Verifies a HistoryProof message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a HistoryProof message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns HistoryProof
         */
        static fromObject(object: { [k: string]: any }): game.HistoryProof;

        /**
         * Creates a plain object from a HistoryProof message. Also converts values to other types if specified.
         * @param message HistoryProof
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.HistoryProof, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this HistoryProof to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for HistoryProof
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace HistoryProof {

        /** Properties of a HistoryProof. */
        interface $Properties {

            /** HistoryProof playerId */
            playerId?: (string|null);

            /** HistoryProof playerName */
            playerName?: (string|null);

            /** HistoryProof text */
            text?: (string|null);

            /** HistoryProof imageUrl */
            imageUrl?: (string|null);

            /** HistoryProof taskText */
            taskText?: (string|null);

            /** HistoryProof status */
            status?: (string|null);

            /** HistoryProof reviewedBy */
            reviewedBy?: (string|null);

            /** HistoryProof reviewedAt */
            reviewedAt?: (number|Long|null);

            /** HistoryProof rejectReason */
            rejectReason?: (string|null);

            /** HistoryProof redoTaskText */
            redoTaskText?: (string|null);

            /** HistoryProof submittedAt */
            submittedAt?: (number|Long|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a HistoryProof. */
        type $Shape = game.HistoryProof.$Properties;
    }

    /**
     * Properties of a RoundHistoryItem.
     * @deprecated Use game.RoundHistoryItem.$Properties instead.
     */
    interface IRoundHistoryItem extends game.RoundHistoryItem.$Properties {
    }

    /** Represents a RoundHistoryItem. */
    class RoundHistoryItem {

        /**
         * Constructs a new RoundHistoryItem.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.RoundHistoryItem.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** RoundHistoryItem id. */
        id: string;

        /** RoundHistoryItem round. */
        round: number;

        /** RoundHistoryItem at. */
        at: (number|Long);

        /** RoundHistoryItem playerA. */
        playerA: string;

        /** RoundHistoryItem playerB. */
        playerB: string;

        /** RoundHistoryItem moveA. */
        moveA: string;

        /** RoundHistoryItem moveB. */
        moveB: string;

        /** RoundHistoryItem result. */
        result: string;

        /** RoundHistoryItem resultLabel. */
        resultLabel: string;

        /** RoundHistoryItem resultText. */
        resultText: string;

        /** RoundHistoryItem gameId. */
        gameId: string;

        /** RoundHistoryItem othelloScore. */
        othelloScore?: (game.OthelloScore.$Properties|null);

        /** RoundHistoryItem othelloBlackSeat. */
        othelloBlackSeat: string;

        /** RoundHistoryItem tictactoeXSeat. */
        tictactoeXSeat: string;

        /** RoundHistoryItem tictactoeLine. */
        tictactoeLine: game.Pos.$Properties[];

        /** RoundHistoryItem ranked. */
        ranked: boolean;

        /** RoundHistoryItem stake. */
        stake: number;

        /** RoundHistoryItem rankMultiplier. */
        rankMultiplier: number;

        /** RoundHistoryItem effectiveStake. */
        effectiveStake: number;

        /** RoundHistoryItem extremeRanked. */
        extremeRanked: boolean;

        /** RoundHistoryItem punishmentName. */
        punishmentName: string;

        /** RoundHistoryItem punishmentDescription. */
        punishmentDescription: string;

        /** RoundHistoryItem punishmentTasks. */
        punishmentTasks: game.PunishmentTask.$Properties[];

        /** RoundHistoryItem punishedNames. */
        punishedNames: string[];

        /** RoundHistoryItem proofs. */
        proofs: game.HistoryProof.$Properties[];

        /** RoundHistoryItem liarsDiceWinnerId. */
        liarsDiceWinnerId: string;

        /** RoundHistoryItem liarsDiceLoserId. */
        liarsDiceLoserId: string;

        /** RoundHistoryItem liarsDiceBidCount. */
        liarsDiceBidCount: number;

        /** RoundHistoryItem liarsDiceBidFace. */
        liarsDiceBidFace: number;

        /** RoundHistoryItem liarsDiceActualCount. */
        liarsDiceActualCount: number;

        /** RoundHistoryItem liarsDiceHands. */
        liarsDiceHands: game.LiarsDiceHandsPair.$Properties[];

        /** RoundHistoryItem liarsDiceHandOrder. */
        liarsDiceHandOrder: string[];

        /** RoundHistoryItem liarsDiceNames. */
        liarsDiceNames: game.StringPair.$Properties[];

        /** RoundHistoryItem gomokuBlackSeat. */
        gomokuBlackSeat: string;

        /** RoundHistoryItem gomokuLine. */
        gomokuLine: game.Pos.$Properties[];

        /** RoundHistoryItem chessWhiteSeat. */
        chessWhiteSeat: string;

        /**
         * Creates a new RoundHistoryItem instance using the specified properties.
         * @param [properties] Properties to set
         * @returns RoundHistoryItem instance
         */
        static create(properties: game.RoundHistoryItem.$Shape): game.RoundHistoryItem & game.RoundHistoryItem.$Shape;
        static create(properties?: game.RoundHistoryItem.$Properties): game.RoundHistoryItem;

        /**
         * Encodes the specified RoundHistoryItem message. Does not implicitly {@link game.RoundHistoryItem.verify|verify} messages.
         * @param message RoundHistoryItem message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.RoundHistoryItem.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified RoundHistoryItem message, length delimited. Does not implicitly {@link game.RoundHistoryItem.verify|verify} messages.
         * @param message RoundHistoryItem message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.RoundHistoryItem.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a RoundHistoryItem message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.RoundHistoryItem & game.RoundHistoryItem.$Shape} RoundHistoryItem
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.RoundHistoryItem & game.RoundHistoryItem.$Shape;

        /**
         * Decodes a RoundHistoryItem message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.RoundHistoryItem & game.RoundHistoryItem.$Shape} RoundHistoryItem
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.RoundHistoryItem & game.RoundHistoryItem.$Shape;

        /**
         * Verifies a RoundHistoryItem message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a RoundHistoryItem message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns RoundHistoryItem
         */
        static fromObject(object: { [k: string]: any }): game.RoundHistoryItem;

        /**
         * Creates a plain object from a RoundHistoryItem message. Also converts values to other types if specified.
         * @param message RoundHistoryItem
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.RoundHistoryItem, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this RoundHistoryItem to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for RoundHistoryItem
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace RoundHistoryItem {

        /** Properties of a RoundHistoryItem. */
        interface $Properties {

            /** RoundHistoryItem id */
            id?: (string|null);

            /** RoundHistoryItem round */
            round?: (number|null);

            /** RoundHistoryItem at */
            at?: (number|Long|null);

            /** RoundHistoryItem playerA */
            playerA?: (string|null);

            /** RoundHistoryItem playerB */
            playerB?: (string|null);

            /** RoundHistoryItem moveA */
            moveA?: (string|null);

            /** RoundHistoryItem moveB */
            moveB?: (string|null);

            /** RoundHistoryItem result */
            result?: (string|null);

            /** RoundHistoryItem resultLabel */
            resultLabel?: (string|null);

            /** RoundHistoryItem resultText */
            resultText?: (string|null);

            /** RoundHistoryItem gameId */
            gameId?: (string|null);

            /** RoundHistoryItem othelloScore */
            othelloScore?: (game.OthelloScore.$Properties|null);

            /** RoundHistoryItem othelloBlackSeat */
            othelloBlackSeat?: (string|null);

            /** RoundHistoryItem tictactoeXSeat */
            tictactoeXSeat?: (string|null);

            /** RoundHistoryItem tictactoeLine */
            tictactoeLine?: (game.Pos.$Properties[]|null);

            /** RoundHistoryItem ranked */
            ranked?: (boolean|null);

            /** RoundHistoryItem stake */
            stake?: (number|null);

            /** RoundHistoryItem rankMultiplier */
            rankMultiplier?: (number|null);

            /** RoundHistoryItem effectiveStake */
            effectiveStake?: (number|null);

            /** RoundHistoryItem extremeRanked */
            extremeRanked?: (boolean|null);

            /** RoundHistoryItem punishmentName */
            punishmentName?: (string|null);

            /** RoundHistoryItem punishmentDescription */
            punishmentDescription?: (string|null);

            /** RoundHistoryItem punishmentTasks */
            punishmentTasks?: (game.PunishmentTask.$Properties[]|null);

            /** RoundHistoryItem punishedNames */
            punishedNames?: (string[]|null);

            /** RoundHistoryItem proofs */
            proofs?: (game.HistoryProof.$Properties[]|null);

            /** RoundHistoryItem liarsDiceWinnerId */
            liarsDiceWinnerId?: (string|null);

            /** RoundHistoryItem liarsDiceLoserId */
            liarsDiceLoserId?: (string|null);

            /** RoundHistoryItem liarsDiceBidCount */
            liarsDiceBidCount?: (number|null);

            /** RoundHistoryItem liarsDiceBidFace */
            liarsDiceBidFace?: (number|null);

            /** RoundHistoryItem liarsDiceActualCount */
            liarsDiceActualCount?: (number|null);

            /** RoundHistoryItem liarsDiceHands */
            liarsDiceHands?: (game.LiarsDiceHandsPair.$Properties[]|null);

            /** RoundHistoryItem liarsDiceHandOrder */
            liarsDiceHandOrder?: (string[]|null);

            /** RoundHistoryItem liarsDiceNames */
            liarsDiceNames?: (game.StringPair.$Properties[]|null);

            /** RoundHistoryItem gomokuBlackSeat */
            gomokuBlackSeat?: (string|null);

            /** RoundHistoryItem gomokuLine */
            gomokuLine?: (game.Pos.$Properties[]|null);

            /** RoundHistoryItem chessWhiteSeat */
            chessWhiteSeat?: (string|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a RoundHistoryItem. */
        type $Shape = game.RoundHistoryItem.$Properties;
    }

    /**
     * Properties of an Int32List.
     * @deprecated Use game.Int32List.$Properties instead.
     */
    interface IInt32List extends game.Int32List.$Properties {
    }

    /** Represents an Int32List. */
    class Int32List {

        /**
         * Constructs a new Int32List.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.Int32List.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** Int32List values. */
        values: number[];

        /**
         * Creates a new Int32List instance using the specified properties.
         * @param [properties] Properties to set
         * @returns Int32List instance
         */
        static create(properties: game.Int32List.$Shape): game.Int32List & game.Int32List.$Shape;
        static create(properties?: game.Int32List.$Properties): game.Int32List;

        /**
         * Encodes the specified Int32List message. Does not implicitly {@link game.Int32List.verify|verify} messages.
         * @param message Int32List message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.Int32List.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified Int32List message, length delimited. Does not implicitly {@link game.Int32List.verify|verify} messages.
         * @param message Int32List message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.Int32List.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes an Int32List message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.Int32List & game.Int32List.$Shape} Int32List
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.Int32List & game.Int32List.$Shape;

        /**
         * Decodes an Int32List message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.Int32List & game.Int32List.$Shape} Int32List
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.Int32List & game.Int32List.$Shape;

        /**
         * Verifies an Int32List message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates an Int32List message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns Int32List
         */
        static fromObject(object: { [k: string]: any }): game.Int32List;

        /**
         * Creates a plain object from an Int32List message. Also converts values to other types if specified.
         * @param message Int32List
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.Int32List, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this Int32List to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for Int32List
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace Int32List {

        /** Properties of an Int32List. */
        interface $Properties {

            /** Int32List values */
            values?: (number[]|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of an Int32List. */
        type $Shape = game.Int32List.$Properties;
    }

    /**
     * Properties of a LiarsDiceHandsPair.
     * @deprecated Use game.LiarsDiceHandsPair.$Properties instead.
     */
    interface ILiarsDiceHandsPair extends game.LiarsDiceHandsPair.$Properties {
    }

    /** Represents a LiarsDiceHandsPair. */
    class LiarsDiceHandsPair {

        /**
         * Constructs a new LiarsDiceHandsPair.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.LiarsDiceHandsPair.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** LiarsDiceHandsPair key. */
        key: string;

        /** LiarsDiceHandsPair value. */
        value?: (game.Int32List.$Properties|null);

        /**
         * Creates a new LiarsDiceHandsPair instance using the specified properties.
         * @param [properties] Properties to set
         * @returns LiarsDiceHandsPair instance
         */
        static create(properties: game.LiarsDiceHandsPair.$Shape): game.LiarsDiceHandsPair & game.LiarsDiceHandsPair.$Shape;
        static create(properties?: game.LiarsDiceHandsPair.$Properties): game.LiarsDiceHandsPair;

        /**
         * Encodes the specified LiarsDiceHandsPair message. Does not implicitly {@link game.LiarsDiceHandsPair.verify|verify} messages.
         * @param message LiarsDiceHandsPair message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.LiarsDiceHandsPair.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified LiarsDiceHandsPair message, length delimited. Does not implicitly {@link game.LiarsDiceHandsPair.verify|verify} messages.
         * @param message LiarsDiceHandsPair message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.LiarsDiceHandsPair.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a LiarsDiceHandsPair message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.LiarsDiceHandsPair & game.LiarsDiceHandsPair.$Shape} LiarsDiceHandsPair
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.LiarsDiceHandsPair & game.LiarsDiceHandsPair.$Shape;

        /**
         * Decodes a LiarsDiceHandsPair message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.LiarsDiceHandsPair & game.LiarsDiceHandsPair.$Shape} LiarsDiceHandsPair
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.LiarsDiceHandsPair & game.LiarsDiceHandsPair.$Shape;

        /**
         * Verifies a LiarsDiceHandsPair message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a LiarsDiceHandsPair message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns LiarsDiceHandsPair
         */
        static fromObject(object: { [k: string]: any }): game.LiarsDiceHandsPair;

        /**
         * Creates a plain object from a LiarsDiceHandsPair message. Also converts values to other types if specified.
         * @param message LiarsDiceHandsPair
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.LiarsDiceHandsPair, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this LiarsDiceHandsPair to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for LiarsDiceHandsPair
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace LiarsDiceHandsPair {

        /** Properties of a LiarsDiceHandsPair. */
        interface $Properties {

            /** LiarsDiceHandsPair key */
            key?: (string|null);

            /** LiarsDiceHandsPair value */
            value?: (game.Int32List.$Properties|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a LiarsDiceHandsPair. */
        type $Shape = game.LiarsDiceHandsPair.$Properties;
    }

    /**
     * Properties of a LiarsDiceBid.
     * @deprecated Use game.LiarsDiceBid.$Properties instead.
     */
    interface ILiarsDiceBid extends game.LiarsDiceBid.$Properties {
    }

    /** Represents a LiarsDiceBid. */
    class LiarsDiceBid {

        /**
         * Constructs a new LiarsDiceBid.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.LiarsDiceBid.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** LiarsDiceBid playerId. */
        playerId: string;

        /** LiarsDiceBid count. */
        count: number;

        /** LiarsDiceBid face. */
        face: number;

        /** LiarsDiceBid at. */
        at: (number|Long);

        /**
         * Creates a new LiarsDiceBid instance using the specified properties.
         * @param [properties] Properties to set
         * @returns LiarsDiceBid instance
         */
        static create(properties: game.LiarsDiceBid.$Shape): game.LiarsDiceBid & game.LiarsDiceBid.$Shape;
        static create(properties?: game.LiarsDiceBid.$Properties): game.LiarsDiceBid;

        /**
         * Encodes the specified LiarsDiceBid message. Does not implicitly {@link game.LiarsDiceBid.verify|verify} messages.
         * @param message LiarsDiceBid message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.LiarsDiceBid.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified LiarsDiceBid message, length delimited. Does not implicitly {@link game.LiarsDiceBid.verify|verify} messages.
         * @param message LiarsDiceBid message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.LiarsDiceBid.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a LiarsDiceBid message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.LiarsDiceBid & game.LiarsDiceBid.$Shape} LiarsDiceBid
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.LiarsDiceBid & game.LiarsDiceBid.$Shape;

        /**
         * Decodes a LiarsDiceBid message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.LiarsDiceBid & game.LiarsDiceBid.$Shape} LiarsDiceBid
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.LiarsDiceBid & game.LiarsDiceBid.$Shape;

        /**
         * Verifies a LiarsDiceBid message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a LiarsDiceBid message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns LiarsDiceBid
         */
        static fromObject(object: { [k: string]: any }): game.LiarsDiceBid;

        /**
         * Creates a plain object from a LiarsDiceBid message. Also converts values to other types if specified.
         * @param message LiarsDiceBid
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.LiarsDiceBid, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this LiarsDiceBid to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for LiarsDiceBid
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace LiarsDiceBid {

        /** Properties of a LiarsDiceBid. */
        interface $Properties {

            /** LiarsDiceBid playerId */
            playerId?: (string|null);

            /** LiarsDiceBid count */
            count?: (number|null);

            /** LiarsDiceBid face */
            face?: (number|null);

            /** LiarsDiceBid at */
            at?: (number|Long|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a LiarsDiceBid. */
        type $Shape = game.LiarsDiceBid.$Properties;
    }

    /**
     * Properties of a LiarsDiceState.
     * @deprecated Use game.LiarsDiceState.$Properties instead.
     */
    interface ILiarsDiceState extends game.LiarsDiceState.$Properties {
    }

    /** Represents a LiarsDiceState. */
    class LiarsDiceState {

        /**
         * Constructs a new LiarsDiceState.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.LiarsDiceState.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** LiarsDiceState participantIds. */
        participantIds: string[];

        /** LiarsDiceState readyPlayerIds. */
        readyPlayerIds: string[];

        /** LiarsDiceState diceCounts. */
        diceCounts: game.IntPair.$Properties[];

        /** LiarsDiceState currentTurn. */
        currentTurn: string;

        /** LiarsDiceState currentBid. */
        currentBid?: (game.LiarsDiceBid.$Properties|null);

        /** LiarsDiceState bidHistory. */
        bidHistory: game.LiarsDiceBid.$Properties[];

        /** LiarsDiceState onesWildDisabled. */
        onesWildDisabled: boolean;

        /** LiarsDiceState roundNumber. */
        roundNumber: number;

        /** LiarsDiceState ended. */
        ended: boolean;

        /** LiarsDiceState winnerId. */
        winnerId: string;

        /** LiarsDiceState loserId. */
        loserId: string;

        /** LiarsDiceState revealedHands. */
        revealedHands: game.LiarsDiceHandsPair.$Properties[];

        /** LiarsDiceState actualCount. */
        actualCount: number;

        /** LiarsDiceState minPlayers. */
        minPlayers: number;

        /** LiarsDiceState maxPlayers. */
        maxPlayers: number;

        /**
         * Creates a new LiarsDiceState instance using the specified properties.
         * @param [properties] Properties to set
         * @returns LiarsDiceState instance
         */
        static create(properties: game.LiarsDiceState.$Shape): game.LiarsDiceState & game.LiarsDiceState.$Shape;
        static create(properties?: game.LiarsDiceState.$Properties): game.LiarsDiceState;

        /**
         * Encodes the specified LiarsDiceState message. Does not implicitly {@link game.LiarsDiceState.verify|verify} messages.
         * @param message LiarsDiceState message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.LiarsDiceState.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified LiarsDiceState message, length delimited. Does not implicitly {@link game.LiarsDiceState.verify|verify} messages.
         * @param message LiarsDiceState message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.LiarsDiceState.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a LiarsDiceState message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.LiarsDiceState & game.LiarsDiceState.$Shape} LiarsDiceState
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.LiarsDiceState & game.LiarsDiceState.$Shape;

        /**
         * Decodes a LiarsDiceState message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.LiarsDiceState & game.LiarsDiceState.$Shape} LiarsDiceState
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.LiarsDiceState & game.LiarsDiceState.$Shape;

        /**
         * Verifies a LiarsDiceState message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a LiarsDiceState message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns LiarsDiceState
         */
        static fromObject(object: { [k: string]: any }): game.LiarsDiceState;

        /**
         * Creates a plain object from a LiarsDiceState message. Also converts values to other types if specified.
         * @param message LiarsDiceState
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.LiarsDiceState, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this LiarsDiceState to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for LiarsDiceState
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace LiarsDiceState {

        /** Properties of a LiarsDiceState. */
        interface $Properties {

            /** LiarsDiceState participantIds */
            participantIds?: (string[]|null);

            /** LiarsDiceState readyPlayerIds */
            readyPlayerIds?: (string[]|null);

            /** LiarsDiceState diceCounts */
            diceCounts?: (game.IntPair.$Properties[]|null);

            /** LiarsDiceState currentTurn */
            currentTurn?: (string|null);

            /** LiarsDiceState currentBid */
            currentBid?: (game.LiarsDiceBid.$Properties|null);

            /** LiarsDiceState bidHistory */
            bidHistory?: (game.LiarsDiceBid.$Properties[]|null);

            /** LiarsDiceState onesWildDisabled */
            onesWildDisabled?: (boolean|null);

            /** LiarsDiceState roundNumber */
            roundNumber?: (number|null);

            /** LiarsDiceState ended */
            ended?: (boolean|null);

            /** LiarsDiceState winnerId */
            winnerId?: (string|null);

            /** LiarsDiceState loserId */
            loserId?: (string|null);

            /** LiarsDiceState revealedHands */
            revealedHands?: (game.LiarsDiceHandsPair.$Properties[]|null);

            /** LiarsDiceState actualCount */
            actualCount?: (number|null);

            /** LiarsDiceState minPlayers */
            minPlayers?: (number|null);

            /** LiarsDiceState maxPlayers */
            maxPlayers?: (number|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a LiarsDiceState. */
        type $Shape = game.LiarsDiceState.$Properties;
    }

    /**
     * Properties of an OthelloPendingSettlement.
     * @deprecated Use game.OthelloPendingSettlement.$Properties instead.
     */
    interface IOthelloPendingSettlement extends game.OthelloPendingSettlement.$Properties {
    }

    /** Represents an OthelloPendingSettlement. */
    class OthelloPendingSettlement {

        /**
         * Constructs a new OthelloPendingSettlement.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.OthelloPendingSettlement.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** OthelloPendingSettlement id. */
        id: string;

        /** OthelloPendingSettlement seat. */
        seat: string;

        /** OthelloPendingSettlement opponentSeat. */
        opponentSeat: string;

        /** OthelloPendingSettlement flips. */
        flips: number;

        /** OthelloPendingSettlement stake. */
        stake: number;

        /** OthelloPendingSettlement nextTurn. */
        nextTurn: string;

        /** OthelloPendingSettlement expiresAt. */
        expiresAt: (number|Long);

        /** OthelloPendingSettlement forced. */
        forced: string;

        /** OthelloPendingSettlement resolvedAs. */
        resolvedAs: string;

        /** OthelloPendingSettlement forcedByMasterName. */
        forcedByMasterName: string;

        /**
         * Creates a new OthelloPendingSettlement instance using the specified properties.
         * @param [properties] Properties to set
         * @returns OthelloPendingSettlement instance
         */
        static create(properties: game.OthelloPendingSettlement.$Shape): game.OthelloPendingSettlement & game.OthelloPendingSettlement.$Shape;
        static create(properties?: game.OthelloPendingSettlement.$Properties): game.OthelloPendingSettlement;

        /**
         * Encodes the specified OthelloPendingSettlement message. Does not implicitly {@link game.OthelloPendingSettlement.verify|verify} messages.
         * @param message OthelloPendingSettlement message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.OthelloPendingSettlement.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified OthelloPendingSettlement message, length delimited. Does not implicitly {@link game.OthelloPendingSettlement.verify|verify} messages.
         * @param message OthelloPendingSettlement message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.OthelloPendingSettlement.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes an OthelloPendingSettlement message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.OthelloPendingSettlement & game.OthelloPendingSettlement.$Shape} OthelloPendingSettlement
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.OthelloPendingSettlement & game.OthelloPendingSettlement.$Shape;

        /**
         * Decodes an OthelloPendingSettlement message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.OthelloPendingSettlement & game.OthelloPendingSettlement.$Shape} OthelloPendingSettlement
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.OthelloPendingSettlement & game.OthelloPendingSettlement.$Shape;

        /**
         * Verifies an OthelloPendingSettlement message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates an OthelloPendingSettlement message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns OthelloPendingSettlement
         */
        static fromObject(object: { [k: string]: any }): game.OthelloPendingSettlement;

        /**
         * Creates a plain object from an OthelloPendingSettlement message. Also converts values to other types if specified.
         * @param message OthelloPendingSettlement
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.OthelloPendingSettlement, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this OthelloPendingSettlement to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for OthelloPendingSettlement
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace OthelloPendingSettlement {

        /** Properties of an OthelloPendingSettlement. */
        interface $Properties {

            /** OthelloPendingSettlement id */
            id?: (string|null);

            /** OthelloPendingSettlement seat */
            seat?: (string|null);

            /** OthelloPendingSettlement opponentSeat */
            opponentSeat?: (string|null);

            /** OthelloPendingSettlement flips */
            flips?: (number|null);

            /** OthelloPendingSettlement stake */
            stake?: (number|null);

            /** OthelloPendingSettlement nextTurn */
            nextTurn?: (string|null);

            /** OthelloPendingSettlement expiresAt */
            expiresAt?: (number|Long|null);

            /** OthelloPendingSettlement forced */
            forced?: (string|null);

            /** OthelloPendingSettlement resolvedAs */
            resolvedAs?: (string|null);

            /** OthelloPendingSettlement forcedByMasterName */
            forcedByMasterName?: (string|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of an OthelloPendingSettlement. */
        type $Shape = game.OthelloPendingSettlement.$Properties;
    }

    /**
     * Properties of an OthelloSurrenderRequest.
     * @deprecated Use game.OthelloSurrenderRequest.$Properties instead.
     */
    interface IOthelloSurrenderRequest extends game.OthelloSurrenderRequest.$Properties {
    }

    /** Represents an OthelloSurrenderRequest. */
    class OthelloSurrenderRequest {

        /**
         * Constructs a new OthelloSurrenderRequest.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.OthelloSurrenderRequest.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** OthelloSurrenderRequest fromSeat. */
        fromSeat: string;

        /** OthelloSurrenderRequest toSeat. */
        toSeat: string;

        /** OthelloSurrenderRequest createdAt. */
        createdAt: (number|Long);

        /**
         * Creates a new OthelloSurrenderRequest instance using the specified properties.
         * @param [properties] Properties to set
         * @returns OthelloSurrenderRequest instance
         */
        static create(properties: game.OthelloSurrenderRequest.$Shape): game.OthelloSurrenderRequest & game.OthelloSurrenderRequest.$Shape;
        static create(properties?: game.OthelloSurrenderRequest.$Properties): game.OthelloSurrenderRequest;

        /**
         * Encodes the specified OthelloSurrenderRequest message. Does not implicitly {@link game.OthelloSurrenderRequest.verify|verify} messages.
         * @param message OthelloSurrenderRequest message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.OthelloSurrenderRequest.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified OthelloSurrenderRequest message, length delimited. Does not implicitly {@link game.OthelloSurrenderRequest.verify|verify} messages.
         * @param message OthelloSurrenderRequest message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.OthelloSurrenderRequest.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes an OthelloSurrenderRequest message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.OthelloSurrenderRequest & game.OthelloSurrenderRequest.$Shape} OthelloSurrenderRequest
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.OthelloSurrenderRequest & game.OthelloSurrenderRequest.$Shape;

        /**
         * Decodes an OthelloSurrenderRequest message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.OthelloSurrenderRequest & game.OthelloSurrenderRequest.$Shape} OthelloSurrenderRequest
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.OthelloSurrenderRequest & game.OthelloSurrenderRequest.$Shape;

        /**
         * Verifies an OthelloSurrenderRequest message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates an OthelloSurrenderRequest message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns OthelloSurrenderRequest
         */
        static fromObject(object: { [k: string]: any }): game.OthelloSurrenderRequest;

        /**
         * Creates a plain object from an OthelloSurrenderRequest message. Also converts values to other types if specified.
         * @param message OthelloSurrenderRequest
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.OthelloSurrenderRequest, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this OthelloSurrenderRequest to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for OthelloSurrenderRequest
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace OthelloSurrenderRequest {

        /** Properties of an OthelloSurrenderRequest. */
        interface $Properties {

            /** OthelloSurrenderRequest fromSeat */
            fromSeat?: (string|null);

            /** OthelloSurrenderRequest toSeat */
            toSeat?: (string|null);

            /** OthelloSurrenderRequest createdAt */
            createdAt?: (number|Long|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of an OthelloSurrenderRequest. */
        type $Shape = game.OthelloSurrenderRequest.$Properties;
    }

    /**
     * Properties of an OthelloUndoRequest.
     * @deprecated Use game.OthelloUndoRequest.$Properties instead.
     */
    interface IOthelloUndoRequest extends game.OthelloUndoRequest.$Properties {
    }

    /** Represents an OthelloUndoRequest. */
    class OthelloUndoRequest {

        /**
         * Constructs a new OthelloUndoRequest.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.OthelloUndoRequest.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** OthelloUndoRequest fromSeat. */
        fromSeat: string;

        /** OthelloUndoRequest toSeat. */
        toSeat: string;

        /** OthelloUndoRequest createdAt. */
        createdAt: (number|Long);

        /** OthelloUndoRequest expiresAt. */
        expiresAt: (number|Long);

        /**
         * Creates a new OthelloUndoRequest instance using the specified properties.
         * @param [properties] Properties to set
         * @returns OthelloUndoRequest instance
         */
        static create(properties: game.OthelloUndoRequest.$Shape): game.OthelloUndoRequest & game.OthelloUndoRequest.$Shape;
        static create(properties?: game.OthelloUndoRequest.$Properties): game.OthelloUndoRequest;

        /**
         * Encodes the specified OthelloUndoRequest message. Does not implicitly {@link game.OthelloUndoRequest.verify|verify} messages.
         * @param message OthelloUndoRequest message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.OthelloUndoRequest.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified OthelloUndoRequest message, length delimited. Does not implicitly {@link game.OthelloUndoRequest.verify|verify} messages.
         * @param message OthelloUndoRequest message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.OthelloUndoRequest.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes an OthelloUndoRequest message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.OthelloUndoRequest & game.OthelloUndoRequest.$Shape} OthelloUndoRequest
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.OthelloUndoRequest & game.OthelloUndoRequest.$Shape;

        /**
         * Decodes an OthelloUndoRequest message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.OthelloUndoRequest & game.OthelloUndoRequest.$Shape} OthelloUndoRequest
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.OthelloUndoRequest & game.OthelloUndoRequest.$Shape;

        /**
         * Verifies an OthelloUndoRequest message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates an OthelloUndoRequest message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns OthelloUndoRequest
         */
        static fromObject(object: { [k: string]: any }): game.OthelloUndoRequest;

        /**
         * Creates a plain object from an OthelloUndoRequest message. Also converts values to other types if specified.
         * @param message OthelloUndoRequest
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.OthelloUndoRequest, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this OthelloUndoRequest to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for OthelloUndoRequest
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace OthelloUndoRequest {

        /** Properties of an OthelloUndoRequest. */
        interface $Properties {

            /** OthelloUndoRequest fromSeat */
            fromSeat?: (string|null);

            /** OthelloUndoRequest toSeat */
            toSeat?: (string|null);

            /** OthelloUndoRequest createdAt */
            createdAt?: (number|Long|null);

            /** OthelloUndoRequest expiresAt */
            expiresAt?: (number|Long|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of an OthelloUndoRequest. */
        type $Shape = game.OthelloUndoRequest.$Properties;
    }

    /**
     * Properties of a BoardRow.
     * @deprecated Use game.BoardRow.$Properties instead.
     */
    interface IBoardRow extends game.BoardRow.$Properties {
    }

    /** Represents a BoardRow. */
    class BoardRow {

        /**
         * Constructs a new BoardRow.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.BoardRow.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** BoardRow cells. */
        cells: string[];

        /**
         * Creates a new BoardRow instance using the specified properties.
         * @param [properties] Properties to set
         * @returns BoardRow instance
         */
        static create(properties: game.BoardRow.$Shape): game.BoardRow & game.BoardRow.$Shape;
        static create(properties?: game.BoardRow.$Properties): game.BoardRow;

        /**
         * Encodes the specified BoardRow message. Does not implicitly {@link game.BoardRow.verify|verify} messages.
         * @param message BoardRow message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.BoardRow.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified BoardRow message, length delimited. Does not implicitly {@link game.BoardRow.verify|verify} messages.
         * @param message BoardRow message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.BoardRow.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a BoardRow message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.BoardRow & game.BoardRow.$Shape} BoardRow
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.BoardRow & game.BoardRow.$Shape;

        /**
         * Decodes a BoardRow message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.BoardRow & game.BoardRow.$Shape} BoardRow
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.BoardRow & game.BoardRow.$Shape;

        /**
         * Verifies a BoardRow message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a BoardRow message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns BoardRow
         */
        static fromObject(object: { [k: string]: any }): game.BoardRow;

        /**
         * Creates a plain object from a BoardRow message. Also converts values to other types if specified.
         * @param message BoardRow
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.BoardRow, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this BoardRow to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for BoardRow
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace BoardRow {

        /** Properties of a BoardRow. */
        interface $Properties {

            /** BoardRow cells */
            cells?: (string[]|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a BoardRow. */
        type $Shape = game.BoardRow.$Properties;
    }

    /**
     * Properties of an IntPair.
     * @deprecated Use game.IntPair.$Properties instead.
     */
    interface IIntPair extends game.IntPair.$Properties {
    }

    /** Represents an IntPair. */
    class IntPair {

        /**
         * Constructs a new IntPair.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.IntPair.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** IntPair key. */
        key: string;

        /** IntPair value. */
        value: number;

        /**
         * Creates a new IntPair instance using the specified properties.
         * @param [properties] Properties to set
         * @returns IntPair instance
         */
        static create(properties: game.IntPair.$Shape): game.IntPair & game.IntPair.$Shape;
        static create(properties?: game.IntPair.$Properties): game.IntPair;

        /**
         * Encodes the specified IntPair message. Does not implicitly {@link game.IntPair.verify|verify} messages.
         * @param message IntPair message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.IntPair.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified IntPair message, length delimited. Does not implicitly {@link game.IntPair.verify|verify} messages.
         * @param message IntPair message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.IntPair.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes an IntPair message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.IntPair & game.IntPair.$Shape} IntPair
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.IntPair & game.IntPair.$Shape;

        /**
         * Decodes an IntPair message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.IntPair & game.IntPair.$Shape} IntPair
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.IntPair & game.IntPair.$Shape;

        /**
         * Verifies an IntPair message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates an IntPair message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns IntPair
         */
        static fromObject(object: { [k: string]: any }): game.IntPair;

        /**
         * Creates a plain object from an IntPair message. Also converts values to other types if specified.
         * @param message IntPair
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.IntPair, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this IntPair to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for IntPair
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace IntPair {

        /** Properties of an IntPair. */
        interface $Properties {

            /** IntPair key */
            key?: (string|null);

            /** IntPair value */
            value?: (number|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of an IntPair. */
        type $Shape = game.IntPair.$Properties;
    }

    /**
     * Properties of an OthelloState.
     * @deprecated Use game.OthelloState.$Properties instead.
     */
    interface IOthelloState extends game.OthelloState.$Properties {
    }

    /** Represents an OthelloState. */
    class OthelloState {

        /**
         * Constructs a new OthelloState.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.OthelloState.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** OthelloState board. */
        board: game.BoardRow.$Properties[];

        /** OthelloState turn. */
        turn: string;

        /** OthelloState blackSeat. */
        blackSeat: string;

        /** OthelloState legalMoves. */
        legalMoves: game.Pos.$Properties[];

        /** OthelloState passCount. */
        passCount: number;

        /** OthelloState blackCount. */
        blackCount: number;

        /** OthelloState whiteCount. */
        whiteCount: number;

        /** OthelloState rankedDelta. */
        rankedDelta: game.IntPair.$Properties[];

        /** OthelloState settlementEvents. */
        settlementEvents: string[];

        /** OthelloState pendingSettlement. */
        pendingSettlement?: (game.OthelloPendingSettlement.$Properties|null);

        /** OthelloState surrenderRequest. */
        surrenderRequest?: (game.OthelloSurrenderRequest.$Properties|null);

        /** OthelloState ended. */
        ended: boolean;

        /** OthelloState winner. */
        winner: string;

        /** OthelloState moveDeadlineAt. */
        moveDeadlineAt: (number|Long);

        /** OthelloState clockDeadlineAt. */
        clockDeadlineAt: (number|Long);

        /** OthelloState clockRemaining. */
        clockRemaining: game.IntPair.$Properties[];

        /** OthelloState undoCount. */
        undoCount: game.IntPair.$Properties[];

        /** OthelloState undoRequest. */
        undoRequest?: (game.OthelloUndoRequest.$Properties|null);

        /**
         * Creates a new OthelloState instance using the specified properties.
         * @param [properties] Properties to set
         * @returns OthelloState instance
         */
        static create(properties: game.OthelloState.$Shape): game.OthelloState & game.OthelloState.$Shape;
        static create(properties?: game.OthelloState.$Properties): game.OthelloState;

        /**
         * Encodes the specified OthelloState message. Does not implicitly {@link game.OthelloState.verify|verify} messages.
         * @param message OthelloState message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.OthelloState.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified OthelloState message, length delimited. Does not implicitly {@link game.OthelloState.verify|verify} messages.
         * @param message OthelloState message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.OthelloState.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes an OthelloState message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.OthelloState & game.OthelloState.$Shape} OthelloState
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.OthelloState & game.OthelloState.$Shape;

        /**
         * Decodes an OthelloState message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.OthelloState & game.OthelloState.$Shape} OthelloState
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.OthelloState & game.OthelloState.$Shape;

        /**
         * Verifies an OthelloState message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates an OthelloState message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns OthelloState
         */
        static fromObject(object: { [k: string]: any }): game.OthelloState;

        /**
         * Creates a plain object from an OthelloState message. Also converts values to other types if specified.
         * @param message OthelloState
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.OthelloState, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this OthelloState to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for OthelloState
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace OthelloState {

        /** Properties of an OthelloState. */
        interface $Properties {

            /** OthelloState board */
            board?: (game.BoardRow.$Properties[]|null);

            /** OthelloState turn */
            turn?: (string|null);

            /** OthelloState blackSeat */
            blackSeat?: (string|null);

            /** OthelloState legalMoves */
            legalMoves?: (game.Pos.$Properties[]|null);

            /** OthelloState passCount */
            passCount?: (number|null);

            /** OthelloState blackCount */
            blackCount?: (number|null);

            /** OthelloState whiteCount */
            whiteCount?: (number|null);

            /** OthelloState rankedDelta */
            rankedDelta?: (game.IntPair.$Properties[]|null);

            /** OthelloState settlementEvents */
            settlementEvents?: (string[]|null);

            /** OthelloState pendingSettlement */
            pendingSettlement?: (game.OthelloPendingSettlement.$Properties|null);

            /** OthelloState surrenderRequest */
            surrenderRequest?: (game.OthelloSurrenderRequest.$Properties|null);

            /** OthelloState ended */
            ended?: (boolean|null);

            /** OthelloState winner */
            winner?: (string|null);

            /** OthelloState moveDeadlineAt */
            moveDeadlineAt?: (number|Long|null);

            /** OthelloState clockDeadlineAt */
            clockDeadlineAt?: (number|Long|null);

            /** OthelloState clockRemaining */
            clockRemaining?: (game.IntPair.$Properties[]|null);

            /** OthelloState undoCount */
            undoCount?: (game.IntPair.$Properties[]|null);

            /** OthelloState undoRequest */
            undoRequest?: (game.OthelloUndoRequest.$Properties|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of an OthelloState. */
        type $Shape = game.OthelloState.$Properties;
    }

    /**
     * Properties of a TicTacToeGiveawayPrompt.
     * @deprecated Use game.TicTacToeGiveawayPrompt.$Properties instead.
     */
    interface ITicTacToeGiveawayPrompt extends game.TicTacToeGiveawayPrompt.$Properties {
    }

    /** Represents a TicTacToeGiveawayPrompt. */
    class TicTacToeGiveawayPrompt {

        /**
         * Constructs a new TicTacToeGiveawayPrompt.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.TicTacToeGiveawayPrompt.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** TicTacToeGiveawayPrompt seat. */
        seat: string;

        /** TicTacToeGiveawayPrompt forced. */
        forced: boolean;

        /** TicTacToeGiveawayPrompt startedAt. */
        startedAt: (number|Long);

        /** TicTacToeGiveawayPrompt expiresAt. */
        expiresAt: (number|Long);

        /**
         * Creates a new TicTacToeGiveawayPrompt instance using the specified properties.
         * @param [properties] Properties to set
         * @returns TicTacToeGiveawayPrompt instance
         */
        static create(properties: game.TicTacToeGiveawayPrompt.$Shape): game.TicTacToeGiveawayPrompt & game.TicTacToeGiveawayPrompt.$Shape;
        static create(properties?: game.TicTacToeGiveawayPrompt.$Properties): game.TicTacToeGiveawayPrompt;

        /**
         * Encodes the specified TicTacToeGiveawayPrompt message. Does not implicitly {@link game.TicTacToeGiveawayPrompt.verify|verify} messages.
         * @param message TicTacToeGiveawayPrompt message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.TicTacToeGiveawayPrompt.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified TicTacToeGiveawayPrompt message, length delimited. Does not implicitly {@link game.TicTacToeGiveawayPrompt.verify|verify} messages.
         * @param message TicTacToeGiveawayPrompt message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.TicTacToeGiveawayPrompt.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a TicTacToeGiveawayPrompt message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.TicTacToeGiveawayPrompt & game.TicTacToeGiveawayPrompt.$Shape} TicTacToeGiveawayPrompt
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.TicTacToeGiveawayPrompt & game.TicTacToeGiveawayPrompt.$Shape;

        /**
         * Decodes a TicTacToeGiveawayPrompt message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.TicTacToeGiveawayPrompt & game.TicTacToeGiveawayPrompt.$Shape} TicTacToeGiveawayPrompt
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.TicTacToeGiveawayPrompt & game.TicTacToeGiveawayPrompt.$Shape;

        /**
         * Verifies a TicTacToeGiveawayPrompt message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a TicTacToeGiveawayPrompt message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns TicTacToeGiveawayPrompt
         */
        static fromObject(object: { [k: string]: any }): game.TicTacToeGiveawayPrompt;

        /**
         * Creates a plain object from a TicTacToeGiveawayPrompt message. Also converts values to other types if specified.
         * @param message TicTacToeGiveawayPrompt
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.TicTacToeGiveawayPrompt, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this TicTacToeGiveawayPrompt to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for TicTacToeGiveawayPrompt
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace TicTacToeGiveawayPrompt {

        /** Properties of a TicTacToeGiveawayPrompt. */
        interface $Properties {

            /** TicTacToeGiveawayPrompt seat */
            seat?: (string|null);

            /** TicTacToeGiveawayPrompt forced */
            forced?: (boolean|null);

            /** TicTacToeGiveawayPrompt startedAt */
            startedAt?: (number|Long|null);

            /** TicTacToeGiveawayPrompt expiresAt */
            expiresAt?: (number|Long|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a TicTacToeGiveawayPrompt. */
        type $Shape = game.TicTacToeGiveawayPrompt.$Properties;
    }

    /**
     * Properties of a TicTacToeState.
     * @deprecated Use game.TicTacToeState.$Properties instead.
     */
    interface ITicTacToeState extends game.TicTacToeState.$Properties {
    }

    /** Represents a TicTacToeState. */
    class TicTacToeState {

        /**
         * Constructs a new TicTacToeState.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.TicTacToeState.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** TicTacToeState board. */
        board: game.BoardRow.$Properties[];

        /** TicTacToeState turn. */
        turn: string;

        /** TicTacToeState xSeat. */
        xSeat: string;

        /** TicTacToeState moveCount. */
        moveCount: number;

        /** TicTacToeState giveawayPrompt. */
        giveawayPrompt?: (game.TicTacToeGiveawayPrompt.$Properties|null);

        /** TicTacToeState winningLine. */
        winningLine: game.Pos.$Properties[];

        /** TicTacToeState rankedDelta. */
        rankedDelta: game.IntPair.$Properties[];

        /** TicTacToeState ended. */
        ended: boolean;

        /** TicTacToeState winner. */
        winner: string;

        /**
         * Creates a new TicTacToeState instance using the specified properties.
         * @param [properties] Properties to set
         * @returns TicTacToeState instance
         */
        static create(properties: game.TicTacToeState.$Shape): game.TicTacToeState & game.TicTacToeState.$Shape;
        static create(properties?: game.TicTacToeState.$Properties): game.TicTacToeState;

        /**
         * Encodes the specified TicTacToeState message. Does not implicitly {@link game.TicTacToeState.verify|verify} messages.
         * @param message TicTacToeState message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.TicTacToeState.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified TicTacToeState message, length delimited. Does not implicitly {@link game.TicTacToeState.verify|verify} messages.
         * @param message TicTacToeState message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.TicTacToeState.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a TicTacToeState message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.TicTacToeState & game.TicTacToeState.$Shape} TicTacToeState
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.TicTacToeState & game.TicTacToeState.$Shape;

        /**
         * Decodes a TicTacToeState message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.TicTacToeState & game.TicTacToeState.$Shape} TicTacToeState
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.TicTacToeState & game.TicTacToeState.$Shape;

        /**
         * Verifies a TicTacToeState message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a TicTacToeState message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns TicTacToeState
         */
        static fromObject(object: { [k: string]: any }): game.TicTacToeState;

        /**
         * Creates a plain object from a TicTacToeState message. Also converts values to other types if specified.
         * @param message TicTacToeState
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.TicTacToeState, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this TicTacToeState to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for TicTacToeState
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace TicTacToeState {

        /** Properties of a TicTacToeState. */
        interface $Properties {

            /** TicTacToeState board */
            board?: (game.BoardRow.$Properties[]|null);

            /** TicTacToeState turn */
            turn?: (string|null);

            /** TicTacToeState xSeat */
            xSeat?: (string|null);

            /** TicTacToeState moveCount */
            moveCount?: (number|null);

            /** TicTacToeState giveawayPrompt */
            giveawayPrompt?: (game.TicTacToeGiveawayPrompt.$Properties|null);

            /** TicTacToeState winningLine */
            winningLine?: (game.Pos.$Properties[]|null);

            /** TicTacToeState rankedDelta */
            rankedDelta?: (game.IntPair.$Properties[]|null);

            /** TicTacToeState ended */
            ended?: (boolean|null);

            /** TicTacToeState winner */
            winner?: (string|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a TicTacToeState. */
        type $Shape = game.TicTacToeState.$Properties;
    }

    /**
     * Properties of a BoolPair.
     * @deprecated Use game.BoolPair.$Properties instead.
     */
    interface IBoolPair extends game.BoolPair.$Properties {
    }

    /** Represents a BoolPair. */
    class BoolPair {

        /**
         * Constructs a new BoolPair.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.BoolPair.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** BoolPair key. */
        key: string;

        /** BoolPair value. */
        value: boolean;

        /**
         * Creates a new BoolPair instance using the specified properties.
         * @param [properties] Properties to set
         * @returns BoolPair instance
         */
        static create(properties: game.BoolPair.$Shape): game.BoolPair & game.BoolPair.$Shape;
        static create(properties?: game.BoolPair.$Properties): game.BoolPair;

        /**
         * Encodes the specified BoolPair message. Does not implicitly {@link game.BoolPair.verify|verify} messages.
         * @param message BoolPair message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.BoolPair.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified BoolPair message, length delimited. Does not implicitly {@link game.BoolPair.verify|verify} messages.
         * @param message BoolPair message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.BoolPair.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a BoolPair message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.BoolPair & game.BoolPair.$Shape} BoolPair
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.BoolPair & game.BoolPair.$Shape;

        /**
         * Decodes a BoolPair message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.BoolPair & game.BoolPair.$Shape} BoolPair
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.BoolPair & game.BoolPair.$Shape;

        /**
         * Verifies a BoolPair message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a BoolPair message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns BoolPair
         */
        static fromObject(object: { [k: string]: any }): game.BoolPair;

        /**
         * Creates a plain object from a BoolPair message. Also converts values to other types if specified.
         * @param message BoolPair
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.BoolPair, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this BoolPair to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for BoolPair
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace BoolPair {

        /** Properties of a BoolPair. */
        interface $Properties {

            /** BoolPair key */
            key?: (string|null);

            /** BoolPair value */
            value?: (boolean|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a BoolPair. */
        type $Shape = game.BoolPair.$Properties;
    }

    /**
     * Properties of a StringPair.
     * @deprecated Use game.StringPair.$Properties instead.
     */
    interface IStringPair extends game.StringPair.$Properties {
    }

    /** Represents a StringPair. */
    class StringPair {

        /**
         * Constructs a new StringPair.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.StringPair.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** StringPair key. */
        key: string;

        /** StringPair value. */
        value: string;

        /**
         * Creates a new StringPair instance using the specified properties.
         * @param [properties] Properties to set
         * @returns StringPair instance
         */
        static create(properties: game.StringPair.$Shape): game.StringPair & game.StringPair.$Shape;
        static create(properties?: game.StringPair.$Properties): game.StringPair;

        /**
         * Encodes the specified StringPair message. Does not implicitly {@link game.StringPair.verify|verify} messages.
         * @param message StringPair message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.StringPair.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified StringPair message, length delimited. Does not implicitly {@link game.StringPair.verify|verify} messages.
         * @param message StringPair message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.StringPair.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a StringPair message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.StringPair & game.StringPair.$Shape} StringPair
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.StringPair & game.StringPair.$Shape;

        /**
         * Decodes a StringPair message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.StringPair & game.StringPair.$Shape} StringPair
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.StringPair & game.StringPair.$Shape;

        /**
         * Verifies a StringPair message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a StringPair message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns StringPair
         */
        static fromObject(object: { [k: string]: any }): game.StringPair;

        /**
         * Creates a plain object from a StringPair message. Also converts values to other types if specified.
         * @param message StringPair
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.StringPair, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this StringPair to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for StringPair
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace StringPair {

        /** Properties of a StringPair. */
        interface $Properties {

            /** StringPair key */
            key?: (string|null);

            /** StringPair value */
            value?: (string|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a StringPair. */
        type $Shape = game.StringPair.$Properties;
    }

    /**
     * Properties of a SeatOccupantPair.
     * @deprecated Use game.SeatOccupantPair.$Properties instead.
     */
    interface ISeatOccupantPair extends game.SeatOccupantPair.$Properties {
    }

    /** Represents a SeatOccupantPair. */
    class SeatOccupantPair {

        /**
         * Constructs a new SeatOccupantPair.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.SeatOccupantPair.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** SeatOccupantPair key. */
        key: string;

        /** SeatOccupantPair value. */
        value?: (game.SeatOccupant.$Properties|null);

        /**
         * Creates a new SeatOccupantPair instance using the specified properties.
         * @param [properties] Properties to set
         * @returns SeatOccupantPair instance
         */
        static create(properties: game.SeatOccupantPair.$Shape): game.SeatOccupantPair & game.SeatOccupantPair.$Shape;
        static create(properties?: game.SeatOccupantPair.$Properties): game.SeatOccupantPair;

        /**
         * Encodes the specified SeatOccupantPair message. Does not implicitly {@link game.SeatOccupantPair.verify|verify} messages.
         * @param message SeatOccupantPair message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.SeatOccupantPair.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified SeatOccupantPair message, length delimited. Does not implicitly {@link game.SeatOccupantPair.verify|verify} messages.
         * @param message SeatOccupantPair message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.SeatOccupantPair.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a SeatOccupantPair message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.SeatOccupantPair & game.SeatOccupantPair.$Shape} SeatOccupantPair
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.SeatOccupantPair & game.SeatOccupantPair.$Shape;

        /**
         * Decodes a SeatOccupantPair message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.SeatOccupantPair & game.SeatOccupantPair.$Shape} SeatOccupantPair
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.SeatOccupantPair & game.SeatOccupantPair.$Shape;

        /**
         * Verifies a SeatOccupantPair message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a SeatOccupantPair message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns SeatOccupantPair
         */
        static fromObject(object: { [k: string]: any }): game.SeatOccupantPair;

        /**
         * Creates a plain object from a SeatOccupantPair message. Also converts values to other types if specified.
         * @param message SeatOccupantPair
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.SeatOccupantPair, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this SeatOccupantPair to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for SeatOccupantPair
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace SeatOccupantPair {

        /** Properties of a SeatOccupantPair. */
        interface $Properties {

            /** SeatOccupantPair key */
            key?: (string|null);

            /** SeatOccupantPair value */
            value?: (game.SeatOccupant.$Properties|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a SeatOccupantPair. */
        type $Shape = {
          key?: string|null;
          value?: game.SeatOccupant.$Shape|null;
          $unknowns?: Uint8Array[];
        };
    }

    /**
     * Properties of a SeatStatsPair.
     * @deprecated Use game.SeatStatsPair.$Properties instead.
     */
    interface ISeatStatsPair extends game.SeatStatsPair.$Properties {
    }

    /** Represents a SeatStatsPair. */
    class SeatStatsPair {

        /**
         * Constructs a new SeatStatsPair.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.SeatStatsPair.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** SeatStatsPair key. */
        key: string;

        /** SeatStatsPair value. */
        value?: (game.SeatStats.$Properties|null);

        /**
         * Creates a new SeatStatsPair instance using the specified properties.
         * @param [properties] Properties to set
         * @returns SeatStatsPair instance
         */
        static create(properties: game.SeatStatsPair.$Shape): game.SeatStatsPair & game.SeatStatsPair.$Shape;
        static create(properties?: game.SeatStatsPair.$Properties): game.SeatStatsPair;

        /**
         * Encodes the specified SeatStatsPair message. Does not implicitly {@link game.SeatStatsPair.verify|verify} messages.
         * @param message SeatStatsPair message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.SeatStatsPair.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified SeatStatsPair message, length delimited. Does not implicitly {@link game.SeatStatsPair.verify|verify} messages.
         * @param message SeatStatsPair message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.SeatStatsPair.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a SeatStatsPair message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.SeatStatsPair & game.SeatStatsPair.$Shape} SeatStatsPair
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.SeatStatsPair & game.SeatStatsPair.$Shape;

        /**
         * Decodes a SeatStatsPair message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.SeatStatsPair & game.SeatStatsPair.$Shape} SeatStatsPair
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.SeatStatsPair & game.SeatStatsPair.$Shape;

        /**
         * Verifies a SeatStatsPair message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a SeatStatsPair message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns SeatStatsPair
         */
        static fromObject(object: { [k: string]: any }): game.SeatStatsPair;

        /**
         * Creates a plain object from a SeatStatsPair message. Also converts values to other types if specified.
         * @param message SeatStatsPair
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.SeatStatsPair, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this SeatStatsPair to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for SeatStatsPair
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace SeatStatsPair {

        /** Properties of a SeatStatsPair. */
        interface $Properties {

            /** SeatStatsPair key */
            key?: (string|null);

            /** SeatStatsPair value */
            value?: (game.SeatStats.$Properties|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a SeatStatsPair. */
        type $Shape = game.SeatStatsPair.$Properties;
    }

    /**
     * Properties of a RoomSnapshot.
     * @deprecated Use game.RoomSnapshot.$Properties instead.
     */
    interface IRoomSnapshot extends game.RoomSnapshot.$Properties {
    }

    /** Represents a RoomSnapshot. */
    class RoomSnapshot {

        /**
         * Constructs a new RoomSnapshot.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.RoomSnapshot.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** RoomSnapshot id. */
        id: string;

        /** RoomSnapshot updatedAt. */
        updatedAt: (number|Long);

        /** RoomSnapshot settings. */
        settings?: (game.RoomSettings.$Properties|null);

        /** RoomSnapshot status. */
        status: string;

        /** RoomSnapshot phase. */
        phase: string;

        /** RoomSnapshot seats. */
        seats: game.SeatOccupantPair.$Properties[];

        /** RoomSnapshot spectators. */
        spectators: game.PublicPlayer.$Properties[];

        /** RoomSnapshot ready. */
        ready: game.BoolPair.$Properties[];

        /** RoomSnapshot choices. */
        choices: game.StringPair.$Properties[];

        /** RoomSnapshot revealedChoices. */
        revealedChoices: game.StringPair.$Properties[];

        /** RoomSnapshot othello. */
        othello?: (game.OthelloState.$Properties|null);

        /** RoomSnapshot tictactoe. */
        tictactoe?: (game.TicTacToeState.$Properties|null);

        /** RoomSnapshot liarsDice. */
        liarsDice?: (game.LiarsDiceState.$Properties|null);

        /** RoomSnapshot gomoku. */
        gomoku?: (game.GomokuState.$Properties|null);

        /** RoomSnapshot jungle. */
        jungle?: (game.JungleState.$Properties|null);

        /** RoomSnapshot chess. */
        chess?: (game.ChessState.$Properties|null);

        /** RoomSnapshot resultText. */
        resultText: string;

        /** RoomSnapshot punishedPlayerIds. */
        punishedPlayerIds: string[];

        /** RoomSnapshot proofs. */
        proofs: game.PunishmentProof.$Properties[];

        /** RoomSnapshot score. */
        score: game.IntPair.$Properties[];

        /** RoomSnapshot seatedScore. */
        seatedScore: game.IntPair.$Properties[];

        /** RoomSnapshot seatStats. */
        seatStats: game.SeatStatsPair.$Properties[];

        /** RoomSnapshot roundHistory. */
        roundHistory: game.RoundHistoryItem.$Properties[];

        /** RoomSnapshot roundHistoryTotal. */
        roundHistoryTotal: number;

        /** RoomSnapshot chat. */
        chat: game.ChatMessage.$Properties[];

        /** RoomSnapshot forgiveAdvantageTargetId. */
        forgiveAdvantageTargetId: string;

        /** RoomSnapshot forgiveAdvantageBeneficiaryId. */
        forgiveAdvantageBeneficiaryId: string;

        /**
         * Creates a new RoomSnapshot instance using the specified properties.
         * @param [properties] Properties to set
         * @returns RoomSnapshot instance
         */
        static create(properties: game.RoomSnapshot.$Shape): game.RoomSnapshot & game.RoomSnapshot.$Shape;
        static create(properties?: game.RoomSnapshot.$Properties): game.RoomSnapshot;

        /**
         * Encodes the specified RoomSnapshot message. Does not implicitly {@link game.RoomSnapshot.verify|verify} messages.
         * @param message RoomSnapshot message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.RoomSnapshot.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified RoomSnapshot message, length delimited. Does not implicitly {@link game.RoomSnapshot.verify|verify} messages.
         * @param message RoomSnapshot message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.RoomSnapshot.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a RoomSnapshot message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.RoomSnapshot & game.RoomSnapshot.$Shape} RoomSnapshot
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.RoomSnapshot & game.RoomSnapshot.$Shape;

        /**
         * Decodes a RoomSnapshot message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.RoomSnapshot & game.RoomSnapshot.$Shape} RoomSnapshot
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.RoomSnapshot & game.RoomSnapshot.$Shape;

        /**
         * Verifies a RoomSnapshot message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a RoomSnapshot message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns RoomSnapshot
         */
        static fromObject(object: { [k: string]: any }): game.RoomSnapshot;

        /**
         * Creates a plain object from a RoomSnapshot message. Also converts values to other types if specified.
         * @param message RoomSnapshot
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.RoomSnapshot, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this RoomSnapshot to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for RoomSnapshot
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace RoomSnapshot {

        /** Properties of a RoomSnapshot. */
        interface $Properties {

            /** RoomSnapshot id */
            id?: (string|null);

            /** RoomSnapshot updatedAt */
            updatedAt?: (number|Long|null);

            /** RoomSnapshot settings */
            settings?: (game.RoomSettings.$Properties|null);

            /** RoomSnapshot status */
            status?: (string|null);

            /** RoomSnapshot phase */
            phase?: (string|null);

            /** RoomSnapshot seats */
            seats?: (game.SeatOccupantPair.$Properties[]|null);

            /** RoomSnapshot spectators */
            spectators?: (game.PublicPlayer.$Properties[]|null);

            /** RoomSnapshot ready */
            ready?: (game.BoolPair.$Properties[]|null);

            /** RoomSnapshot choices */
            choices?: (game.StringPair.$Properties[]|null);

            /** RoomSnapshot revealedChoices */
            revealedChoices?: (game.StringPair.$Properties[]|null);

            /** RoomSnapshot othello */
            othello?: (game.OthelloState.$Properties|null);

            /** RoomSnapshot tictactoe */
            tictactoe?: (game.TicTacToeState.$Properties|null);

            /** RoomSnapshot liarsDice */
            liarsDice?: (game.LiarsDiceState.$Properties|null);

            /** RoomSnapshot gomoku */
            gomoku?: (game.GomokuState.$Properties|null);

            /** RoomSnapshot jungle */
            jungle?: (game.JungleState.$Properties|null);

            /** RoomSnapshot chess */
            chess?: (game.ChessState.$Properties|null);

            /** RoomSnapshot resultText */
            resultText?: (string|null);

            /** RoomSnapshot punishedPlayerIds */
            punishedPlayerIds?: (string[]|null);

            /** RoomSnapshot proofs */
            proofs?: (game.PunishmentProof.$Properties[]|null);

            /** RoomSnapshot score */
            score?: (game.IntPair.$Properties[]|null);

            /** RoomSnapshot seatedScore */
            seatedScore?: (game.IntPair.$Properties[]|null);

            /** RoomSnapshot seatStats */
            seatStats?: (game.SeatStatsPair.$Properties[]|null);

            /** RoomSnapshot roundHistory */
            roundHistory?: (game.RoundHistoryItem.$Properties[]|null);

            /** RoomSnapshot roundHistoryTotal */
            roundHistoryTotal?: (number|null);

            /** RoomSnapshot chat */
            chat?: (game.ChatMessage.$Properties[]|null);

            /** RoomSnapshot forgiveAdvantageTargetId */
            forgiveAdvantageTargetId?: (string|null);

            /** RoomSnapshot forgiveAdvantageBeneficiaryId */
            forgiveAdvantageBeneficiaryId?: (string|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a RoomSnapshot. */
        type $Shape = {
          id?: string|null;
          updatedAt?: number|Long|null;
          settings?: game.RoomSettings.$Shape|null;
          status?: string|null;
          phase?: string|null;
          seats?: game.SeatOccupantPair.$Shape[]|null;
          spectators?: game.PublicPlayer.$Shape[]|null;
          ready?: game.BoolPair.$Shape[]|null;
          choices?: game.StringPair.$Shape[]|null;
          revealedChoices?: game.StringPair.$Shape[]|null;
          othello?: game.OthelloState.$Shape|null;
          tictactoe?: game.TicTacToeState.$Shape|null;
          liarsDice?: game.LiarsDiceState.$Shape|null;
          gomoku?: game.GomokuState.$Shape|null;
          jungle?: game.JungleState.$Shape|null;
          chess?: game.ChessState.$Shape|null;
          resultText?: string|null;
          punishedPlayerIds?: string[]|null;
          proofs?: game.PunishmentProof.$Shape[]|null;
          score?: game.IntPair.$Shape[]|null;
          seatedScore?: game.IntPair.$Shape[]|null;
          seatStats?: game.SeatStatsPair.$Shape[]|null;
          roundHistory?: game.RoundHistoryItem.$Shape[]|null;
          roundHistoryTotal?: number|null;
          chat?: game.ChatMessage.$Shape[]|null;
          forgiveAdvantageTargetId?: string|null;
          forgiveAdvantageBeneficiaryId?: string|null;
          $unknowns?: Uint8Array[];
        };
    }

    /**
     * Properties of a ServerStats.
     * @deprecated Use game.ServerStats.$Properties instead.
     */
    interface IServerStats extends game.ServerStats.$Properties {
    }

    /** Represents a ServerStats. */
    class ServerStats {

        /**
         * Constructs a new ServerStats.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.ServerStats.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** ServerStats startedAt. */
        startedAt: (number|Long);

        /** ServerStats roomBroadcasts. */
        roomBroadcasts: number;

        /** ServerStats lobbyBroadcasts. */
        lobbyBroadcasts: number;

        /** ServerStats disconnects. */
        disconnects: number;

        /** ServerStats reconnects. */
        reconnects: number;

        /** ServerStats lastRoomSnapshotBytes. */
        lastRoomSnapshotBytes: number;

        /** ServerStats lastLobbySnapshotBytes. */
        lastLobbySnapshotBytes: number;

        /** ServerStats recentRoomBroadcasts. */
        recentRoomBroadcasts: number;

        /** ServerStats recentLobbyBroadcasts. */
        recentLobbyBroadcasts: number;

        /** ServerStats averageRoomSnapshotBytes. */
        averageRoomSnapshotBytes: number;

        /** ServerStats averageLobbySnapshotBytes. */
        averageLobbySnapshotBytes: number;

        /**
         * Creates a new ServerStats instance using the specified properties.
         * @param [properties] Properties to set
         * @returns ServerStats instance
         */
        static create(properties: game.ServerStats.$Shape): game.ServerStats & game.ServerStats.$Shape;
        static create(properties?: game.ServerStats.$Properties): game.ServerStats;

        /**
         * Encodes the specified ServerStats message. Does not implicitly {@link game.ServerStats.verify|verify} messages.
         * @param message ServerStats message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.ServerStats.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified ServerStats message, length delimited. Does not implicitly {@link game.ServerStats.verify|verify} messages.
         * @param message ServerStats message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.ServerStats.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a ServerStats message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.ServerStats & game.ServerStats.$Shape} ServerStats
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.ServerStats & game.ServerStats.$Shape;

        /**
         * Decodes a ServerStats message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.ServerStats & game.ServerStats.$Shape} ServerStats
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.ServerStats & game.ServerStats.$Shape;

        /**
         * Verifies a ServerStats message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a ServerStats message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns ServerStats
         */
        static fromObject(object: { [k: string]: any }): game.ServerStats;

        /**
         * Creates a plain object from a ServerStats message. Also converts values to other types if specified.
         * @param message ServerStats
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.ServerStats, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this ServerStats to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for ServerStats
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace ServerStats {

        /** Properties of a ServerStats. */
        interface $Properties {

            /** ServerStats startedAt */
            startedAt?: (number|Long|null);

            /** ServerStats roomBroadcasts */
            roomBroadcasts?: (number|null);

            /** ServerStats lobbyBroadcasts */
            lobbyBroadcasts?: (number|null);

            /** ServerStats disconnects */
            disconnects?: (number|null);

            /** ServerStats reconnects */
            reconnects?: (number|null);

            /** ServerStats lastRoomSnapshotBytes */
            lastRoomSnapshotBytes?: (number|null);

            /** ServerStats lastLobbySnapshotBytes */
            lastLobbySnapshotBytes?: (number|null);

            /** ServerStats recentRoomBroadcasts */
            recentRoomBroadcasts?: (number|null);

            /** ServerStats recentLobbyBroadcasts */
            recentLobbyBroadcasts?: (number|null);

            /** ServerStats averageRoomSnapshotBytes */
            averageRoomSnapshotBytes?: (number|null);

            /** ServerStats averageLobbySnapshotBytes */
            averageLobbySnapshotBytes?: (number|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a ServerStats. */
        type $Shape = game.ServerStats.$Properties;
    }

    /**
     * Properties of a VersusSeat.
     * @deprecated Use game.VersusSeat.$Properties instead.
     */
    interface IVersusSeat extends game.VersusSeat.$Properties {
    }

    /** Represents a VersusSeat. */
    class VersusSeat {

        /**
         * Constructs a new VersusSeat.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.VersusSeat.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** VersusSeat player. */
        player?: (game.PublicPlayer.$Properties|null);

        /**
         * Creates a new VersusSeat instance using the specified properties.
         * @param [properties] Properties to set
         * @returns VersusSeat instance
         */
        static create(properties: game.VersusSeat.$Shape): game.VersusSeat & game.VersusSeat.$Shape;
        static create(properties?: game.VersusSeat.$Properties): game.VersusSeat;

        /**
         * Encodes the specified VersusSeat message. Does not implicitly {@link game.VersusSeat.verify|verify} messages.
         * @param message VersusSeat message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.VersusSeat.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified VersusSeat message, length delimited. Does not implicitly {@link game.VersusSeat.verify|verify} messages.
         * @param message VersusSeat message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.VersusSeat.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a VersusSeat message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.VersusSeat & game.VersusSeat.$Shape} VersusSeat
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.VersusSeat & game.VersusSeat.$Shape;

        /**
         * Decodes a VersusSeat message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.VersusSeat & game.VersusSeat.$Shape} VersusSeat
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.VersusSeat & game.VersusSeat.$Shape;

        /**
         * Verifies a VersusSeat message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a VersusSeat message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns VersusSeat
         */
        static fromObject(object: { [k: string]: any }): game.VersusSeat;

        /**
         * Creates a plain object from a VersusSeat message. Also converts values to other types if specified.
         * @param message VersusSeat
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.VersusSeat, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this VersusSeat to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for VersusSeat
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace VersusSeat {

        /** Properties of a VersusSeat. */
        interface $Properties {

            /** VersusSeat player */
            player?: (game.PublicPlayer.$Properties|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a VersusSeat. */
        type $Shape = game.VersusSeat.$Properties;
    }

    /**
     * Properties of a VersusPair.
     * @deprecated Use game.VersusPair.$Properties instead.
     */
    interface IVersusPair extends game.VersusPair.$Properties {
    }

    /** Represents a VersusPair. */
    class VersusPair {

        /**
         * Constructs a new VersusPair.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.VersusPair.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** VersusPair key. */
        key: string;

        /** VersusPair value. */
        value?: (game.VersusSeat.$Properties|null);

        /**
         * Creates a new VersusPair instance using the specified properties.
         * @param [properties] Properties to set
         * @returns VersusPair instance
         */
        static create(properties: game.VersusPair.$Shape): game.VersusPair & game.VersusPair.$Shape;
        static create(properties?: game.VersusPair.$Properties): game.VersusPair;

        /**
         * Encodes the specified VersusPair message. Does not implicitly {@link game.VersusPair.verify|verify} messages.
         * @param message VersusPair message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.VersusPair.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified VersusPair message, length delimited. Does not implicitly {@link game.VersusPair.verify|verify} messages.
         * @param message VersusPair message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.VersusPair.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a VersusPair message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.VersusPair & game.VersusPair.$Shape} VersusPair
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.VersusPair & game.VersusPair.$Shape;

        /**
         * Decodes a VersusPair message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.VersusPair & game.VersusPair.$Shape} VersusPair
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.VersusPair & game.VersusPair.$Shape;

        /**
         * Verifies a VersusPair message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a VersusPair message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns VersusPair
         */
        static fromObject(object: { [k: string]: any }): game.VersusPair;

        /**
         * Creates a plain object from a VersusPair message. Also converts values to other types if specified.
         * @param message VersusPair
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.VersusPair, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this VersusPair to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for VersusPair
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace VersusPair {

        /** Properties of a VersusPair. */
        interface $Properties {

            /** VersusPair key */
            key?: (string|null);

            /** VersusPair value */
            value?: (game.VersusSeat.$Properties|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a VersusPair. */
        type $Shape = game.VersusPair.$Properties;
    }

    /**
     * Properties of a LobbyRoomInfo.
     * @deprecated Use game.LobbyRoomInfo.$Properties instead.
     */
    interface ILobbyRoomInfo extends game.LobbyRoomInfo.$Properties {
    }

    /** Represents a LobbyRoomInfo. */
    class LobbyRoomInfo {

        /**
         * Constructs a new LobbyRoomInfo.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.LobbyRoomInfo.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** LobbyRoomInfo id. */
        id: string;

        /** LobbyRoomInfo gameId. */
        gameId: string;

        /** LobbyRoomInfo name. */
        name: string;

        /** LobbyRoomInfo hasPassword. */
        hasPassword: boolean;

        /** LobbyRoomInfo players. */
        players: number;

        /** LobbyRoomInfo spectators. */
        spectators: number;

        /** LobbyRoomInfo versus. */
        versus: game.VersusPair.$Properties[];

        /** LobbyRoomInfo status. */
        status: string;

        /** LobbyRoomInfo roomBackgroundImage. */
        roomBackgroundImage: string;

        /** LobbyRoomInfo enablePunishment. */
        enablePunishment: boolean;

        /** LobbyRoomInfo punishmentIds. */
        punishmentIds: string[];

        /** LobbyRoomInfo punishmentId. */
        punishmentId: string;

        /** LobbyRoomInfo tieDoublePunish. */
        tieDoublePunish: boolean;

        /** LobbyRoomInfo requireOpponentConfirm. */
        requireOpponentConfirm: boolean;

        /** LobbyRoomInfo enableRanked. */
        enableRanked: boolean;

        /** LobbyRoomInfo stake. */
        stake: number;

        /** LobbyRoomInfo enableRankMultiplier. */
        enableRankMultiplier: boolean;

        /** LobbyRoomInfo rankMultiplier. */
        rankMultiplier: number;

        /** LobbyRoomInfo enableExtremeRanked. */
        enableExtremeRanked: boolean;

        /** LobbyRoomInfo tags. */
        tags: string[];

        /** LobbyRoomInfo liarsDiceMinPlayers. */
        liarsDiceMinPlayers: number;

        /** LobbyRoomInfo liarsDiceMaxPlayers. */
        liarsDiceMaxPlayers: number;

        /** LobbyRoomInfo othelloMoveSeconds. */
        othelloMoveSeconds: number;

        /** LobbyRoomInfo othelloGameMinutes. */
        othelloGameMinutes: number;

        /** LobbyRoomInfo gomokuMoveSeconds. */
        gomokuMoveSeconds: number;

        /** LobbyRoomInfo gomokuGameMinutes. */
        gomokuGameMinutes: number;

        /** LobbyRoomInfo gomokuUndoLimit. */
        gomokuUndoLimit: number;

        /** LobbyRoomInfo jungleMoveSeconds. */
        jungleMoveSeconds: number;

        /** LobbyRoomInfo jungleGameMinutes. */
        jungleGameMinutes: number;

        /** LobbyRoomInfo punishmentSource. */
        punishmentSource: string;

        /** LobbyRoomInfo punishmentTagsIncluded. */
        punishmentTagsIncluded: string[];

        /** LobbyRoomInfo punishmentTagsExcluded. */
        punishmentTagsExcluded: string[];

        /** LobbyRoomInfo punishmentSeriesId. */
        punishmentSeriesId: string;

        /** LobbyRoomInfo chessMoveSeconds. */
        chessMoveSeconds: number;

        /** LobbyRoomInfo chessGameMinutes. */
        chessGameMinutes: number;

        /** LobbyRoomInfo jungleUndoLimit. */
        jungleUndoLimit: number;

        /** LobbyRoomInfo chessUndoLimit. */
        chessUndoLimit: number;

        /** LobbyRoomInfo othelloUndoLimit. */
        othelloUndoLimit: number;

        /**
         * Creates a new LobbyRoomInfo instance using the specified properties.
         * @param [properties] Properties to set
         * @returns LobbyRoomInfo instance
         */
        static create(properties: game.LobbyRoomInfo.$Shape): game.LobbyRoomInfo & game.LobbyRoomInfo.$Shape;
        static create(properties?: game.LobbyRoomInfo.$Properties): game.LobbyRoomInfo;

        /**
         * Encodes the specified LobbyRoomInfo message. Does not implicitly {@link game.LobbyRoomInfo.verify|verify} messages.
         * @param message LobbyRoomInfo message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.LobbyRoomInfo.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified LobbyRoomInfo message, length delimited. Does not implicitly {@link game.LobbyRoomInfo.verify|verify} messages.
         * @param message LobbyRoomInfo message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.LobbyRoomInfo.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a LobbyRoomInfo message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.LobbyRoomInfo & game.LobbyRoomInfo.$Shape} LobbyRoomInfo
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.LobbyRoomInfo & game.LobbyRoomInfo.$Shape;

        /**
         * Decodes a LobbyRoomInfo message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.LobbyRoomInfo & game.LobbyRoomInfo.$Shape} LobbyRoomInfo
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.LobbyRoomInfo & game.LobbyRoomInfo.$Shape;

        /**
         * Verifies a LobbyRoomInfo message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a LobbyRoomInfo message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns LobbyRoomInfo
         */
        static fromObject(object: { [k: string]: any }): game.LobbyRoomInfo;

        /**
         * Creates a plain object from a LobbyRoomInfo message. Also converts values to other types if specified.
         * @param message LobbyRoomInfo
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.LobbyRoomInfo, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this LobbyRoomInfo to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for LobbyRoomInfo
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace LobbyRoomInfo {

        /** Properties of a LobbyRoomInfo. */
        interface $Properties {

            /** LobbyRoomInfo id */
            id?: (string|null);

            /** LobbyRoomInfo gameId */
            gameId?: (string|null);

            /** LobbyRoomInfo name */
            name?: (string|null);

            /** LobbyRoomInfo hasPassword */
            hasPassword?: (boolean|null);

            /** LobbyRoomInfo players */
            players?: (number|null);

            /** LobbyRoomInfo spectators */
            spectators?: (number|null);

            /** LobbyRoomInfo versus */
            versus?: (game.VersusPair.$Properties[]|null);

            /** LobbyRoomInfo status */
            status?: (string|null);

            /** LobbyRoomInfo roomBackgroundImage */
            roomBackgroundImage?: (string|null);

            /** LobbyRoomInfo enablePunishment */
            enablePunishment?: (boolean|null);

            /** LobbyRoomInfo punishmentIds */
            punishmentIds?: (string[]|null);

            /** LobbyRoomInfo punishmentId */
            punishmentId?: (string|null);

            /** LobbyRoomInfo tieDoublePunish */
            tieDoublePunish?: (boolean|null);

            /** LobbyRoomInfo requireOpponentConfirm */
            requireOpponentConfirm?: (boolean|null);

            /** LobbyRoomInfo enableRanked */
            enableRanked?: (boolean|null);

            /** LobbyRoomInfo stake */
            stake?: (number|null);

            /** LobbyRoomInfo enableRankMultiplier */
            enableRankMultiplier?: (boolean|null);

            /** LobbyRoomInfo rankMultiplier */
            rankMultiplier?: (number|null);

            /** LobbyRoomInfo enableExtremeRanked */
            enableExtremeRanked?: (boolean|null);

            /** LobbyRoomInfo tags */
            tags?: (string[]|null);

            /** LobbyRoomInfo liarsDiceMinPlayers */
            liarsDiceMinPlayers?: (number|null);

            /** LobbyRoomInfo liarsDiceMaxPlayers */
            liarsDiceMaxPlayers?: (number|null);

            /** LobbyRoomInfo othelloMoveSeconds */
            othelloMoveSeconds?: (number|null);

            /** LobbyRoomInfo othelloGameMinutes */
            othelloGameMinutes?: (number|null);

            /** LobbyRoomInfo gomokuMoveSeconds */
            gomokuMoveSeconds?: (number|null);

            /** LobbyRoomInfo gomokuGameMinutes */
            gomokuGameMinutes?: (number|null);

            /** LobbyRoomInfo gomokuUndoLimit */
            gomokuUndoLimit?: (number|null);

            /** LobbyRoomInfo jungleMoveSeconds */
            jungleMoveSeconds?: (number|null);

            /** LobbyRoomInfo jungleGameMinutes */
            jungleGameMinutes?: (number|null);

            /** LobbyRoomInfo punishmentSource */
            punishmentSource?: (string|null);

            /** LobbyRoomInfo punishmentTagsIncluded */
            punishmentTagsIncluded?: (string[]|null);

            /** LobbyRoomInfo punishmentTagsExcluded */
            punishmentTagsExcluded?: (string[]|null);

            /** LobbyRoomInfo punishmentSeriesId */
            punishmentSeriesId?: (string|null);

            /** LobbyRoomInfo chessMoveSeconds */
            chessMoveSeconds?: (number|null);

            /** LobbyRoomInfo chessGameMinutes */
            chessGameMinutes?: (number|null);

            /** LobbyRoomInfo jungleUndoLimit */
            jungleUndoLimit?: (number|null);

            /** LobbyRoomInfo chessUndoLimit */
            chessUndoLimit?: (number|null);

            /** LobbyRoomInfo othelloUndoLimit */
            othelloUndoLimit?: (number|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a LobbyRoomInfo. */
        type $Shape = game.LobbyRoomInfo.$Properties;
    }

    /**
     * Properties of a LobbyPlayerEntry.
     * @deprecated Use game.LobbyPlayerEntry.$Properties instead.
     */
    interface ILobbyPlayerEntry extends game.LobbyPlayerEntry.$Properties {
    }

    /** Represents a LobbyPlayerEntry. */
    class LobbyPlayerEntry {

        /**
         * Constructs a new LobbyPlayerEntry.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.LobbyPlayerEntry.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** LobbyPlayerEntry id. */
        id: string;

        /** LobbyPlayerEntry player. */
        player?: (game.LobbyPlayer.$Properties|null);

        /**
         * Creates a new LobbyPlayerEntry instance using the specified properties.
         * @param [properties] Properties to set
         * @returns LobbyPlayerEntry instance
         */
        static create(properties: game.LobbyPlayerEntry.$Shape): game.LobbyPlayerEntry & game.LobbyPlayerEntry.$Shape;
        static create(properties?: game.LobbyPlayerEntry.$Properties): game.LobbyPlayerEntry;

        /**
         * Encodes the specified LobbyPlayerEntry message. Does not implicitly {@link game.LobbyPlayerEntry.verify|verify} messages.
         * @param message LobbyPlayerEntry message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.LobbyPlayerEntry.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified LobbyPlayerEntry message, length delimited. Does not implicitly {@link game.LobbyPlayerEntry.verify|verify} messages.
         * @param message LobbyPlayerEntry message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.LobbyPlayerEntry.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a LobbyPlayerEntry message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.LobbyPlayerEntry & game.LobbyPlayerEntry.$Shape} LobbyPlayerEntry
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.LobbyPlayerEntry & game.LobbyPlayerEntry.$Shape;

        /**
         * Decodes a LobbyPlayerEntry message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.LobbyPlayerEntry & game.LobbyPlayerEntry.$Shape} LobbyPlayerEntry
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.LobbyPlayerEntry & game.LobbyPlayerEntry.$Shape;

        /**
         * Verifies a LobbyPlayerEntry message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a LobbyPlayerEntry message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns LobbyPlayerEntry
         */
        static fromObject(object: { [k: string]: any }): game.LobbyPlayerEntry;

        /**
         * Creates a plain object from a LobbyPlayerEntry message. Also converts values to other types if specified.
         * @param message LobbyPlayerEntry
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.LobbyPlayerEntry, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this LobbyPlayerEntry to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for LobbyPlayerEntry
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace LobbyPlayerEntry {

        /** Properties of a LobbyPlayerEntry. */
        interface $Properties {

            /** LobbyPlayerEntry id */
            id?: (string|null);

            /** LobbyPlayerEntry player */
            player?: (game.LobbyPlayer.$Properties|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a LobbyPlayerEntry. */
        type $Shape = game.LobbyPlayerEntry.$Properties;
    }

    /**
     * Properties of a LobbyRoomEntry.
     * @deprecated Use game.LobbyRoomEntry.$Properties instead.
     */
    interface ILobbyRoomEntry extends game.LobbyRoomEntry.$Properties {
    }

    /** Represents a LobbyRoomEntry. */
    class LobbyRoomEntry {

        /**
         * Constructs a new LobbyRoomEntry.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.LobbyRoomEntry.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** LobbyRoomEntry id. */
        id: string;

        /** LobbyRoomEntry room. */
        room?: (game.LobbyRoomInfo.$Properties|null);

        /**
         * Creates a new LobbyRoomEntry instance using the specified properties.
         * @param [properties] Properties to set
         * @returns LobbyRoomEntry instance
         */
        static create(properties: game.LobbyRoomEntry.$Shape): game.LobbyRoomEntry & game.LobbyRoomEntry.$Shape;
        static create(properties?: game.LobbyRoomEntry.$Properties): game.LobbyRoomEntry;

        /**
         * Encodes the specified LobbyRoomEntry message. Does not implicitly {@link game.LobbyRoomEntry.verify|verify} messages.
         * @param message LobbyRoomEntry message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.LobbyRoomEntry.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified LobbyRoomEntry message, length delimited. Does not implicitly {@link game.LobbyRoomEntry.verify|verify} messages.
         * @param message LobbyRoomEntry message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.LobbyRoomEntry.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a LobbyRoomEntry message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.LobbyRoomEntry & game.LobbyRoomEntry.$Shape} LobbyRoomEntry
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.LobbyRoomEntry & game.LobbyRoomEntry.$Shape;

        /**
         * Decodes a LobbyRoomEntry message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.LobbyRoomEntry & game.LobbyRoomEntry.$Shape} LobbyRoomEntry
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.LobbyRoomEntry & game.LobbyRoomEntry.$Shape;

        /**
         * Verifies a LobbyRoomEntry message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a LobbyRoomEntry message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns LobbyRoomEntry
         */
        static fromObject(object: { [k: string]: any }): game.LobbyRoomEntry;

        /**
         * Creates a plain object from a LobbyRoomEntry message. Also converts values to other types if specified.
         * @param message LobbyRoomEntry
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.LobbyRoomEntry, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this LobbyRoomEntry to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for LobbyRoomEntry
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace LobbyRoomEntry {

        /** Properties of a LobbyRoomEntry. */
        interface $Properties {

            /** LobbyRoomEntry id */
            id?: (string|null);

            /** LobbyRoomEntry room */
            room?: (game.LobbyRoomInfo.$Properties|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a LobbyRoomEntry. */
        type $Shape = game.LobbyRoomEntry.$Properties;
    }

    /**
     * Properties of a PetBondEdge.
     * @deprecated Use game.PetBondEdge.$Properties instead.
     */
    interface IPetBondEdge extends game.PetBondEdge.$Properties {
    }

    /** Represents a PetBondEdge. */
    class PetBondEdge {

        /**
         * Constructs a new PetBondEdge.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.PetBondEdge.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** PetBondEdge masterId. */
        masterId: string;

        /** PetBondEdge petId. */
        petId: string;

        /** PetBondEdge petTitle. */
        petTitle: string;

        /**
         * Creates a new PetBondEdge instance using the specified properties.
         * @param [properties] Properties to set
         * @returns PetBondEdge instance
         */
        static create(properties: game.PetBondEdge.$Shape): game.PetBondEdge & game.PetBondEdge.$Shape;
        static create(properties?: game.PetBondEdge.$Properties): game.PetBondEdge;

        /**
         * Encodes the specified PetBondEdge message. Does not implicitly {@link game.PetBondEdge.verify|verify} messages.
         * @param message PetBondEdge message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.PetBondEdge.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified PetBondEdge message, length delimited. Does not implicitly {@link game.PetBondEdge.verify|verify} messages.
         * @param message PetBondEdge message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.PetBondEdge.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a PetBondEdge message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.PetBondEdge & game.PetBondEdge.$Shape} PetBondEdge
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.PetBondEdge & game.PetBondEdge.$Shape;

        /**
         * Decodes a PetBondEdge message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.PetBondEdge & game.PetBondEdge.$Shape} PetBondEdge
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.PetBondEdge & game.PetBondEdge.$Shape;

        /**
         * Verifies a PetBondEdge message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a PetBondEdge message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns PetBondEdge
         */
        static fromObject(object: { [k: string]: any }): game.PetBondEdge;

        /**
         * Creates a plain object from a PetBondEdge message. Also converts values to other types if specified.
         * @param message PetBondEdge
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.PetBondEdge, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this PetBondEdge to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for PetBondEdge
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace PetBondEdge {

        /** Properties of a PetBondEdge. */
        interface $Properties {

            /** PetBondEdge masterId */
            masterId?: (string|null);

            /** PetBondEdge petId */
            petId?: (string|null);

            /** PetBondEdge petTitle */
            petTitle?: (string|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a PetBondEdge. */
        type $Shape = game.PetBondEdge.$Properties;
    }

    /**
     * Properties of a LobbySnapshot.
     * @deprecated Use game.LobbySnapshot.$Properties instead.
     */
    interface ILobbySnapshot extends game.LobbySnapshot.$Properties {
    }

    /** Represents a LobbySnapshot. */
    class LobbySnapshot {

        /**
         * Constructs a new LobbySnapshot.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.LobbySnapshot.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** LobbySnapshot config. */
        config?: (game.AppConfig.$Properties|null);

        /** LobbySnapshot onlineCount. */
        onlineCount: number;

        /** LobbySnapshot players. */
        players: game.LobbyPlayerEntry.$Properties[];

        /** LobbySnapshot rooms. */
        rooms: game.LobbyRoomEntry.$Properties[];

        /** LobbySnapshot normalLeaderboard. */
        normalLeaderboard: game.LobbyPlayer.$Properties[];

        /** LobbySnapshot rankedLeaderboard. */
        rankedLeaderboard: game.LobbyPlayer.$Properties[];

        /** LobbySnapshot suggestions. */
        suggestions: game.Suggestion.$Properties[];

        /** LobbySnapshot lobbyChat. */
        lobbyChat: game.ChatMessage.$Properties[];

        /** LobbySnapshot serverStats. */
        serverStats?: (game.ServerStats.$Properties|null);

        /** LobbySnapshot petBonds. */
        petBonds: game.PetBondEdge.$Properties[];

        /**
         * Creates a new LobbySnapshot instance using the specified properties.
         * @param [properties] Properties to set
         * @returns LobbySnapshot instance
         */
        static create(properties: game.LobbySnapshot.$Shape): game.LobbySnapshot & game.LobbySnapshot.$Shape;
        static create(properties?: game.LobbySnapshot.$Properties): game.LobbySnapshot;

        /**
         * Encodes the specified LobbySnapshot message. Does not implicitly {@link game.LobbySnapshot.verify|verify} messages.
         * @param message LobbySnapshot message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.LobbySnapshot.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified LobbySnapshot message, length delimited. Does not implicitly {@link game.LobbySnapshot.verify|verify} messages.
         * @param message LobbySnapshot message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.LobbySnapshot.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a LobbySnapshot message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.LobbySnapshot & game.LobbySnapshot.$Shape} LobbySnapshot
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.LobbySnapshot & game.LobbySnapshot.$Shape;

        /**
         * Decodes a LobbySnapshot message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.LobbySnapshot & game.LobbySnapshot.$Shape} LobbySnapshot
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.LobbySnapshot & game.LobbySnapshot.$Shape;

        /**
         * Verifies a LobbySnapshot message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a LobbySnapshot message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns LobbySnapshot
         */
        static fromObject(object: { [k: string]: any }): game.LobbySnapshot;

        /**
         * Creates a plain object from a LobbySnapshot message. Also converts values to other types if specified.
         * @param message LobbySnapshot
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.LobbySnapshot, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this LobbySnapshot to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for LobbySnapshot
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace LobbySnapshot {

        /** Properties of a LobbySnapshot. */
        interface $Properties {

            /** LobbySnapshot config */
            config?: (game.AppConfig.$Properties|null);

            /** LobbySnapshot onlineCount */
            onlineCount?: (number|null);

            /** LobbySnapshot players */
            players?: (game.LobbyPlayerEntry.$Properties[]|null);

            /** LobbySnapshot rooms */
            rooms?: (game.LobbyRoomEntry.$Properties[]|null);

            /** LobbySnapshot normalLeaderboard */
            normalLeaderboard?: (game.LobbyPlayer.$Properties[]|null);

            /** LobbySnapshot rankedLeaderboard */
            rankedLeaderboard?: (game.LobbyPlayer.$Properties[]|null);

            /** LobbySnapshot suggestions */
            suggestions?: (game.Suggestion.$Properties[]|null);

            /** LobbySnapshot lobbyChat */
            lobbyChat?: (game.ChatMessage.$Properties[]|null);

            /** LobbySnapshot serverStats */
            serverStats?: (game.ServerStats.$Properties|null);

            /** LobbySnapshot petBonds */
            petBonds?: (game.PetBondEdge.$Properties[]|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a LobbySnapshot. */
        type $Shape = game.LobbySnapshot.$Properties;
    }

    /**
     * Properties of a RoomNamePool.
     * @deprecated Use game.RoomNamePool.$Properties instead.
     */
    interface IRoomNamePool extends game.RoomNamePool.$Properties {
    }

    /** Represents a RoomNamePool. */
    class RoomNamePool {

        /**
         * Constructs a new RoomNamePool.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.RoomNamePool.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** RoomNamePool adjectives. */
        adjectives: string[];

        /** RoomNamePool subjects. */
        subjects: string[];

        /** RoomNamePool roomWords. */
        roomWords: string[];

        /**
         * Creates a new RoomNamePool instance using the specified properties.
         * @param [properties] Properties to set
         * @returns RoomNamePool instance
         */
        static create(properties: game.RoomNamePool.$Shape): game.RoomNamePool & game.RoomNamePool.$Shape;
        static create(properties?: game.RoomNamePool.$Properties): game.RoomNamePool;

        /**
         * Encodes the specified RoomNamePool message. Does not implicitly {@link game.RoomNamePool.verify|verify} messages.
         * @param message RoomNamePool message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.RoomNamePool.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified RoomNamePool message, length delimited. Does not implicitly {@link game.RoomNamePool.verify|verify} messages.
         * @param message RoomNamePool message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.RoomNamePool.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a RoomNamePool message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.RoomNamePool & game.RoomNamePool.$Shape} RoomNamePool
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.RoomNamePool & game.RoomNamePool.$Shape;

        /**
         * Decodes a RoomNamePool message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.RoomNamePool & game.RoomNamePool.$Shape} RoomNamePool
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.RoomNamePool & game.RoomNamePool.$Shape;

        /**
         * Verifies a RoomNamePool message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a RoomNamePool message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns RoomNamePool
         */
        static fromObject(object: { [k: string]: any }): game.RoomNamePool;

        /**
         * Creates a plain object from a RoomNamePool message. Also converts values to other types if specified.
         * @param message RoomNamePool
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.RoomNamePool, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this RoomNamePool to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for RoomNamePool
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace RoomNamePool {

        /** Properties of a RoomNamePool. */
        interface $Properties {

            /** RoomNamePool adjectives */
            adjectives?: (string[]|null);

            /** RoomNamePool subjects */
            subjects?: (string[]|null);

            /** RoomNamePool roomWords */
            roomWords?: (string[]|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a RoomNamePool. */
        type $Shape = game.RoomNamePool.$Properties;
    }

    /**
     * Properties of a RoomInfoTagStyle.
     * @deprecated Use game.RoomInfoTagStyle.$Properties instead.
     */
    interface IRoomInfoTagStyle extends game.RoomInfoTagStyle.$Properties {
    }

    /** Represents a RoomInfoTagStyle. */
    class RoomInfoTagStyle {

        /**
         * Constructs a new RoomInfoTagStyle.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.RoomInfoTagStyle.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** RoomInfoTagStyle label. */
        label: string;

        /** RoomInfoTagStyle textColor. */
        textColor: string;

        /** RoomInfoTagStyle backgroundColor. */
        backgroundColor: string;

        /** RoomInfoTagStyle borderColor. */
        borderColor: string;

        /**
         * Creates a new RoomInfoTagStyle instance using the specified properties.
         * @param [properties] Properties to set
         * @returns RoomInfoTagStyle instance
         */
        static create(properties: game.RoomInfoTagStyle.$Shape): game.RoomInfoTagStyle & game.RoomInfoTagStyle.$Shape;
        static create(properties?: game.RoomInfoTagStyle.$Properties): game.RoomInfoTagStyle;

        /**
         * Encodes the specified RoomInfoTagStyle message. Does not implicitly {@link game.RoomInfoTagStyle.verify|verify} messages.
         * @param message RoomInfoTagStyle message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.RoomInfoTagStyle.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified RoomInfoTagStyle message, length delimited. Does not implicitly {@link game.RoomInfoTagStyle.verify|verify} messages.
         * @param message RoomInfoTagStyle message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.RoomInfoTagStyle.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a RoomInfoTagStyle message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.RoomInfoTagStyle & game.RoomInfoTagStyle.$Shape} RoomInfoTagStyle
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.RoomInfoTagStyle & game.RoomInfoTagStyle.$Shape;

        /**
         * Decodes a RoomInfoTagStyle message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.RoomInfoTagStyle & game.RoomInfoTagStyle.$Shape} RoomInfoTagStyle
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.RoomInfoTagStyle & game.RoomInfoTagStyle.$Shape;

        /**
         * Verifies a RoomInfoTagStyle message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a RoomInfoTagStyle message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns RoomInfoTagStyle
         */
        static fromObject(object: { [k: string]: any }): game.RoomInfoTagStyle;

        /**
         * Creates a plain object from a RoomInfoTagStyle message. Also converts values to other types if specified.
         * @param message RoomInfoTagStyle
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.RoomInfoTagStyle, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this RoomInfoTagStyle to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for RoomInfoTagStyle
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace RoomInfoTagStyle {

        /** Properties of a RoomInfoTagStyle. */
        interface $Properties {

            /** RoomInfoTagStyle label */
            label?: (string|null);

            /** RoomInfoTagStyle textColor */
            textColor?: (string|null);

            /** RoomInfoTagStyle backgroundColor */
            backgroundColor?: (string|null);

            /** RoomInfoTagStyle borderColor */
            borderColor?: (string|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a RoomInfoTagStyle. */
        type $Shape = game.RoomInfoTagStyle.$Properties;
    }

    /**
     * Properties of a RoomInfoTagEntry.
     * @deprecated Use game.RoomInfoTagEntry.$Properties instead.
     */
    interface IRoomInfoTagEntry extends game.RoomInfoTagEntry.$Properties {
    }

    /** Represents a RoomInfoTagEntry. */
    class RoomInfoTagEntry {

        /**
         * Constructs a new RoomInfoTagEntry.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.RoomInfoTagEntry.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** RoomInfoTagEntry key. */
        key: string;

        /** RoomInfoTagEntry style. */
        style?: (game.RoomInfoTagStyle.$Properties|null);

        /**
         * Creates a new RoomInfoTagEntry instance using the specified properties.
         * @param [properties] Properties to set
         * @returns RoomInfoTagEntry instance
         */
        static create(properties: game.RoomInfoTagEntry.$Shape): game.RoomInfoTagEntry & game.RoomInfoTagEntry.$Shape;
        static create(properties?: game.RoomInfoTagEntry.$Properties): game.RoomInfoTagEntry;

        /**
         * Encodes the specified RoomInfoTagEntry message. Does not implicitly {@link game.RoomInfoTagEntry.verify|verify} messages.
         * @param message RoomInfoTagEntry message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.RoomInfoTagEntry.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified RoomInfoTagEntry message, length delimited. Does not implicitly {@link game.RoomInfoTagEntry.verify|verify} messages.
         * @param message RoomInfoTagEntry message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.RoomInfoTagEntry.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a RoomInfoTagEntry message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.RoomInfoTagEntry & game.RoomInfoTagEntry.$Shape} RoomInfoTagEntry
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.RoomInfoTagEntry & game.RoomInfoTagEntry.$Shape;

        /**
         * Decodes a RoomInfoTagEntry message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.RoomInfoTagEntry & game.RoomInfoTagEntry.$Shape} RoomInfoTagEntry
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.RoomInfoTagEntry & game.RoomInfoTagEntry.$Shape;

        /**
         * Verifies a RoomInfoTagEntry message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a RoomInfoTagEntry message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns RoomInfoTagEntry
         */
        static fromObject(object: { [k: string]: any }): game.RoomInfoTagEntry;

        /**
         * Creates a plain object from a RoomInfoTagEntry message. Also converts values to other types if specified.
         * @param message RoomInfoTagEntry
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.RoomInfoTagEntry, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this RoomInfoTagEntry to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for RoomInfoTagEntry
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace RoomInfoTagEntry {

        /** Properties of a RoomInfoTagEntry. */
        interface $Properties {

            /** RoomInfoTagEntry key */
            key?: (string|null);

            /** RoomInfoTagEntry style */
            style?: (game.RoomInfoTagStyle.$Properties|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a RoomInfoTagEntry. */
        type $Shape = game.RoomInfoTagEntry.$Properties;
    }

    /**
     * Properties of a TitleTagStyleEntry.
     * @deprecated Use game.TitleTagStyleEntry.$Properties instead.
     */
    interface ITitleTagStyleEntry extends game.TitleTagStyleEntry.$Properties {
    }

    /** Represents a TitleTagStyleEntry. */
    class TitleTagStyleEntry {

        /**
         * Constructs a new TitleTagStyleEntry.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.TitleTagStyleEntry.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** TitleTagStyleEntry key. */
        key: string;

        /** TitleTagStyleEntry style. */
        style?: (game.RoomInfoTagStyle.$Properties|null);

        /**
         * Creates a new TitleTagStyleEntry instance using the specified properties.
         * @param [properties] Properties to set
         * @returns TitleTagStyleEntry instance
         */
        static create(properties: game.TitleTagStyleEntry.$Shape): game.TitleTagStyleEntry & game.TitleTagStyleEntry.$Shape;
        static create(properties?: game.TitleTagStyleEntry.$Properties): game.TitleTagStyleEntry;

        /**
         * Encodes the specified TitleTagStyleEntry message. Does not implicitly {@link game.TitleTagStyleEntry.verify|verify} messages.
         * @param message TitleTagStyleEntry message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.TitleTagStyleEntry.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified TitleTagStyleEntry message, length delimited. Does not implicitly {@link game.TitleTagStyleEntry.verify|verify} messages.
         * @param message TitleTagStyleEntry message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.TitleTagStyleEntry.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a TitleTagStyleEntry message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.TitleTagStyleEntry & game.TitleTagStyleEntry.$Shape} TitleTagStyleEntry
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.TitleTagStyleEntry & game.TitleTagStyleEntry.$Shape;

        /**
         * Decodes a TitleTagStyleEntry message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.TitleTagStyleEntry & game.TitleTagStyleEntry.$Shape} TitleTagStyleEntry
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.TitleTagStyleEntry & game.TitleTagStyleEntry.$Shape;

        /**
         * Verifies a TitleTagStyleEntry message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a TitleTagStyleEntry message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns TitleTagStyleEntry
         */
        static fromObject(object: { [k: string]: any }): game.TitleTagStyleEntry;

        /**
         * Creates a plain object from a TitleTagStyleEntry message. Also converts values to other types if specified.
         * @param message TitleTagStyleEntry
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.TitleTagStyleEntry, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this TitleTagStyleEntry to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for TitleTagStyleEntry
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace TitleTagStyleEntry {

        /** Properties of a TitleTagStyleEntry. */
        interface $Properties {

            /** TitleTagStyleEntry key */
            key?: (string|null);

            /** TitleTagStyleEntry style */
            style?: (game.RoomInfoTagStyle.$Properties|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a TitleTagStyleEntry. */
        type $Shape = game.TitleTagStyleEntry.$Properties;
    }

    /**
     * Properties of a PunishmentTaskConfig.
     * @deprecated Use game.PunishmentTaskConfig.$Properties instead.
     */
    interface IPunishmentTaskConfig extends game.PunishmentTaskConfig.$Properties {
    }

    /** Represents a PunishmentTaskConfig. */
    class PunishmentTaskConfig {

        /**
         * Constructs a new PunishmentTaskConfig.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.PunishmentTaskConfig.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** PunishmentTaskConfig id. */
        id: string;

        /** PunishmentTaskConfig name. */
        name: string;

        /** PunishmentTaskConfig variants. */
        variants: game.StringPair.$Properties[];

        /** PunishmentTaskConfig backgroundImages. */
        backgroundImages: string[];

        /** PunishmentTaskConfig backgroundOpacity. */
        backgroundOpacity: number;

        /** PunishmentTaskConfig text. */
        text: string;

        /** PunishmentTaskConfig factionIds. */
        factionIds: string[];

        /** PunishmentTaskConfig order. */
        order: number;

        /** PunishmentTaskConfig tagIds. */
        tagIds: string[];

        /**
         * Creates a new PunishmentTaskConfig instance using the specified properties.
         * @param [properties] Properties to set
         * @returns PunishmentTaskConfig instance
         */
        static create(properties: game.PunishmentTaskConfig.$Shape): game.PunishmentTaskConfig & game.PunishmentTaskConfig.$Shape;
        static create(properties?: game.PunishmentTaskConfig.$Properties): game.PunishmentTaskConfig;

        /**
         * Encodes the specified PunishmentTaskConfig message. Does not implicitly {@link game.PunishmentTaskConfig.verify|verify} messages.
         * @param message PunishmentTaskConfig message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.PunishmentTaskConfig.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified PunishmentTaskConfig message, length delimited. Does not implicitly {@link game.PunishmentTaskConfig.verify|verify} messages.
         * @param message PunishmentTaskConfig message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.PunishmentTaskConfig.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a PunishmentTaskConfig message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.PunishmentTaskConfig & game.PunishmentTaskConfig.$Shape} PunishmentTaskConfig
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.PunishmentTaskConfig & game.PunishmentTaskConfig.$Shape;

        /**
         * Decodes a PunishmentTaskConfig message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.PunishmentTaskConfig & game.PunishmentTaskConfig.$Shape} PunishmentTaskConfig
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.PunishmentTaskConfig & game.PunishmentTaskConfig.$Shape;

        /**
         * Verifies a PunishmentTaskConfig message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a PunishmentTaskConfig message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns PunishmentTaskConfig
         */
        static fromObject(object: { [k: string]: any }): game.PunishmentTaskConfig;

        /**
         * Creates a plain object from a PunishmentTaskConfig message. Also converts values to other types if specified.
         * @param message PunishmentTaskConfig
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.PunishmentTaskConfig, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this PunishmentTaskConfig to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for PunishmentTaskConfig
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace PunishmentTaskConfig {

        /** Properties of a PunishmentTaskConfig. */
        interface $Properties {

            /** PunishmentTaskConfig id */
            id?: (string|null);

            /** PunishmentTaskConfig name */
            name?: (string|null);

            /** PunishmentTaskConfig variants */
            variants?: (game.StringPair.$Properties[]|null);

            /** PunishmentTaskConfig backgroundImages */
            backgroundImages?: (string[]|null);

            /** PunishmentTaskConfig backgroundOpacity */
            backgroundOpacity?: (number|null);

            /** PunishmentTaskConfig text */
            text?: (string|null);

            /** PunishmentTaskConfig factionIds */
            factionIds?: (string[]|null);

            /** PunishmentTaskConfig order */
            order?: (number|null);

            /** PunishmentTaskConfig tagIds */
            tagIds?: (string[]|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a PunishmentTaskConfig. */
        type $Shape = game.PunishmentTaskConfig.$Properties;
    }

    /**
     * Properties of a TitleSegment.
     * @deprecated Use game.TitleSegment.$Properties instead.
     */
    interface ITitleSegment extends game.TitleSegment.$Properties {
    }

    /** Represents a TitleSegment. */
    class TitleSegment {

        /**
         * Constructs a new TitleSegment.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.TitleSegment.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** TitleSegment id. */
        id: string;

        /** TitleSegment minPercent. */
        minPercent: number;

        /** TitleSegment maxPercent. */
        maxPercent: number;

        /** TitleSegment names. */
        names: string[];

        /** TitleSegment factionNames. */
        factionNames: game.TitleSegment.FactionNames.$Properties[];

        /**
         * Creates a new TitleSegment instance using the specified properties.
         * @param [properties] Properties to set
         * @returns TitleSegment instance
         */
        static create(properties: game.TitleSegment.$Shape): game.TitleSegment & game.TitleSegment.$Shape;
        static create(properties?: game.TitleSegment.$Properties): game.TitleSegment;

        /**
         * Encodes the specified TitleSegment message. Does not implicitly {@link game.TitleSegment.verify|verify} messages.
         * @param message TitleSegment message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.TitleSegment.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified TitleSegment message, length delimited. Does not implicitly {@link game.TitleSegment.verify|verify} messages.
         * @param message TitleSegment message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.TitleSegment.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a TitleSegment message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.TitleSegment & game.TitleSegment.$Shape} TitleSegment
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.TitleSegment & game.TitleSegment.$Shape;

        /**
         * Decodes a TitleSegment message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.TitleSegment & game.TitleSegment.$Shape} TitleSegment
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.TitleSegment & game.TitleSegment.$Shape;

        /**
         * Verifies a TitleSegment message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a TitleSegment message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns TitleSegment
         */
        static fromObject(object: { [k: string]: any }): game.TitleSegment;

        /**
         * Creates a plain object from a TitleSegment message. Also converts values to other types if specified.
         * @param message TitleSegment
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.TitleSegment, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this TitleSegment to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for TitleSegment
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace TitleSegment {

        /** Properties of a TitleSegment. */
        interface $Properties {

            /** TitleSegment id */
            id?: (string|null);

            /** TitleSegment minPercent */
            minPercent?: (number|null);

            /** TitleSegment maxPercent */
            maxPercent?: (number|null);

            /** TitleSegment names */
            names?: (string[]|null);

            /** TitleSegment factionNames */
            factionNames?: (game.TitleSegment.FactionNames.$Properties[]|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a TitleSegment. */
        type $Shape = game.TitleSegment.$Properties;

        /**
         * Properties of a FactionNames.
         * @deprecated Use game.TitleSegment.FactionNames.$Properties instead.
         */
        interface IFactionNames extends game.TitleSegment.FactionNames.$Properties {
        }

        /** Represents a FactionNames. */
        class FactionNames {

            /**
             * Constructs a new FactionNames.
             * @param [properties] Properties to set
             */
            constructor(properties?: game.TitleSegment.FactionNames.$Properties);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];

            /** FactionNames factionId. */
            factionId: string;

            /** FactionNames names. */
            names: string[];

            /**
             * Creates a new FactionNames instance using the specified properties.
             * @param [properties] Properties to set
             * @returns FactionNames instance
             */
            static create(properties: game.TitleSegment.FactionNames.$Shape): game.TitleSegment.FactionNames & game.TitleSegment.FactionNames.$Shape;
            static create(properties?: game.TitleSegment.FactionNames.$Properties): game.TitleSegment.FactionNames;

            /**
             * Encodes the specified FactionNames message. Does not implicitly {@link game.TitleSegment.FactionNames.verify|verify} messages.
             * @param message FactionNames message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            static encode(message: game.TitleSegment.FactionNames.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified FactionNames message, length delimited. Does not implicitly {@link game.TitleSegment.FactionNames.verify|verify} messages.
             * @param message FactionNames message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            static encodeDelimited(message: game.TitleSegment.FactionNames.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a FactionNames message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns {game.TitleSegment.FactionNames & game.TitleSegment.FactionNames.$Shape} FactionNames
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.TitleSegment.FactionNames & game.TitleSegment.FactionNames.$Shape;

            /**
             * Decodes a FactionNames message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns {game.TitleSegment.FactionNames & game.TitleSegment.FactionNames.$Shape} FactionNames
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.TitleSegment.FactionNames & game.TitleSegment.FactionNames.$Shape;

            /**
             * Verifies a FactionNames message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates a FactionNames message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns FactionNames
             */
            static fromObject(object: { [k: string]: any }): game.TitleSegment.FactionNames;

            /**
             * Creates a plain object from a FactionNames message. Also converts values to other types if specified.
             * @param message FactionNames
             * @param [options] Conversion options
             * @returns Plain object
             */
            static toObject(message: game.TitleSegment.FactionNames, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this FactionNames to JSON.
             * @returns JSON object
             */
            toJSON(): { [k: string]: any };

            /**
             * Gets the type url for FactionNames
             * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
             * @returns The type url
             */
            static getTypeUrl(prefix?: string): string;
        }

        namespace FactionNames {

            /** Properties of a FactionNames. */
            interface $Properties {

                /** FactionNames factionId */
                factionId?: (string|null);

                /** FactionNames names */
                names?: (string[]|null);

                /** Unknown fields preserved while decoding when enabled */
                $unknowns?: Uint8Array[];
            }

            /** Shape of a FactionNames. */
            type $Shape = game.TitleSegment.FactionNames.$Properties;
        }
    }

    /**
     * Properties of a PunishmentConfig.
     * @deprecated Use game.PunishmentConfig.$Properties instead.
     */
    interface IPunishmentConfig extends game.PunishmentConfig.$Properties {
    }

    /** Represents a PunishmentConfig. */
    class PunishmentConfig {

        /**
         * Constructs a new PunishmentConfig.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.PunishmentConfig.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** PunishmentConfig id. */
        id: string;

        /** PunishmentConfig name. */
        name: string;

        /** PunishmentConfig description. */
        description: string;

        /** PunishmentConfig variants. */
        variants: game.StringPair.$Properties[];

        /** PunishmentConfig tasks. */
        tasks: game.PunishmentTaskConfig.$Properties[];

        /** PunishmentConfig cardImageUrl. */
        cardImageUrl: string;

        /** PunishmentConfig cardImageOpacity. */
        cardImageOpacity: number;

        /** PunishmentConfig roomBackgroundImages. */
        roomBackgroundImages: string[];

        /** PunishmentConfig roomNamePool. */
        roomNamePool?: (game.RoomNamePool.$Properties|null);

        /** PunishmentConfig orderStep. */
        orderStep: number;

        /** PunishmentConfig orderSpread. */
        orderSpread: number;

        /** PunishmentConfig maxDifficultyOvershoot. */
        maxDifficultyOvershoot: number;

        /**
         * Creates a new PunishmentConfig instance using the specified properties.
         * @param [properties] Properties to set
         * @returns PunishmentConfig instance
         */
        static create(properties: game.PunishmentConfig.$Shape): game.PunishmentConfig & game.PunishmentConfig.$Shape;
        static create(properties?: game.PunishmentConfig.$Properties): game.PunishmentConfig;

        /**
         * Encodes the specified PunishmentConfig message. Does not implicitly {@link game.PunishmentConfig.verify|verify} messages.
         * @param message PunishmentConfig message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.PunishmentConfig.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified PunishmentConfig message, length delimited. Does not implicitly {@link game.PunishmentConfig.verify|verify} messages.
         * @param message PunishmentConfig message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.PunishmentConfig.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a PunishmentConfig message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.PunishmentConfig & game.PunishmentConfig.$Shape} PunishmentConfig
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.PunishmentConfig & game.PunishmentConfig.$Shape;

        /**
         * Decodes a PunishmentConfig message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.PunishmentConfig & game.PunishmentConfig.$Shape} PunishmentConfig
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.PunishmentConfig & game.PunishmentConfig.$Shape;

        /**
         * Verifies a PunishmentConfig message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a PunishmentConfig message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns PunishmentConfig
         */
        static fromObject(object: { [k: string]: any }): game.PunishmentConfig;

        /**
         * Creates a plain object from a PunishmentConfig message. Also converts values to other types if specified.
         * @param message PunishmentConfig
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.PunishmentConfig, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this PunishmentConfig to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for PunishmentConfig
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace PunishmentConfig {

        /** Properties of a PunishmentConfig. */
        interface $Properties {

            /** PunishmentConfig id */
            id?: (string|null);

            /** PunishmentConfig name */
            name?: (string|null);

            /** PunishmentConfig description */
            description?: (string|null);

            /** PunishmentConfig variants */
            variants?: (game.StringPair.$Properties[]|null);

            /** PunishmentConfig tasks */
            tasks?: (game.PunishmentTaskConfig.$Properties[]|null);

            /** PunishmentConfig cardImageUrl */
            cardImageUrl?: (string|null);

            /** PunishmentConfig cardImageOpacity */
            cardImageOpacity?: (number|null);

            /** PunishmentConfig roomBackgroundImages */
            roomBackgroundImages?: (string[]|null);

            /** PunishmentConfig roomNamePool */
            roomNamePool?: (game.RoomNamePool.$Properties|null);

            /** PunishmentConfig orderStep */
            orderStep?: (number|null);

            /** PunishmentConfig orderSpread */
            orderSpread?: (number|null);

            /** PunishmentConfig maxDifficultyOvershoot */
            maxDifficultyOvershoot?: (number|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a PunishmentConfig. */
        type $Shape = game.PunishmentConfig.$Properties;
    }

    /**
     * Properties of a PunishmentTagConfig.
     * @deprecated Use game.PunishmentTagConfig.$Properties instead.
     */
    interface IPunishmentTagConfig extends game.PunishmentTagConfig.$Properties {
    }

    /** Represents a PunishmentTagConfig. */
    class PunishmentTagConfig {

        /**
         * Constructs a new PunishmentTagConfig.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.PunishmentTagConfig.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** PunishmentTagConfig id. */
        id: string;

        /** PunishmentTagConfig name. */
        name: string;

        /** PunishmentTagConfig roomNamePool. */
        roomNamePool?: (game.RoomNamePool.$Properties|null);

        /** PunishmentTagConfig roomBackgroundImages. */
        roomBackgroundImages: string[];

        /**
         * Creates a new PunishmentTagConfig instance using the specified properties.
         * @param [properties] Properties to set
         * @returns PunishmentTagConfig instance
         */
        static create(properties: game.PunishmentTagConfig.$Shape): game.PunishmentTagConfig & game.PunishmentTagConfig.$Shape;
        static create(properties?: game.PunishmentTagConfig.$Properties): game.PunishmentTagConfig;

        /**
         * Encodes the specified PunishmentTagConfig message. Does not implicitly {@link game.PunishmentTagConfig.verify|verify} messages.
         * @param message PunishmentTagConfig message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.PunishmentTagConfig.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified PunishmentTagConfig message, length delimited. Does not implicitly {@link game.PunishmentTagConfig.verify|verify} messages.
         * @param message PunishmentTagConfig message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.PunishmentTagConfig.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a PunishmentTagConfig message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.PunishmentTagConfig & game.PunishmentTagConfig.$Shape} PunishmentTagConfig
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.PunishmentTagConfig & game.PunishmentTagConfig.$Shape;

        /**
         * Decodes a PunishmentTagConfig message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.PunishmentTagConfig & game.PunishmentTagConfig.$Shape} PunishmentTagConfig
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.PunishmentTagConfig & game.PunishmentTagConfig.$Shape;

        /**
         * Verifies a PunishmentTagConfig message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a PunishmentTagConfig message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns PunishmentTagConfig
         */
        static fromObject(object: { [k: string]: any }): game.PunishmentTagConfig;

        /**
         * Creates a plain object from a PunishmentTagConfig message. Also converts values to other types if specified.
         * @param message PunishmentTagConfig
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.PunishmentTagConfig, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this PunishmentTagConfig to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for PunishmentTagConfig
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace PunishmentTagConfig {

        /** Properties of a PunishmentTagConfig. */
        interface $Properties {

            /** PunishmentTagConfig id */
            id?: (string|null);

            /** PunishmentTagConfig name */
            name?: (string|null);

            /** PunishmentTagConfig roomNamePool */
            roomNamePool?: (game.RoomNamePool.$Properties|null);

            /** PunishmentTagConfig roomBackgroundImages */
            roomBackgroundImages?: (string[]|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a PunishmentTagConfig. */
        type $Shape = game.PunishmentTagConfig.$Properties;
    }

    /**
     * Properties of a PunishmentRandomSettings.
     * @deprecated Use game.PunishmentRandomSettings.$Properties instead.
     */
    interface IPunishmentRandomSettings extends game.PunishmentRandomSettings.$Properties {
    }

    /** Represents a PunishmentRandomSettings. */
    class PunishmentRandomSettings {

        /**
         * Constructs a new PunishmentRandomSettings.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.PunishmentRandomSettings.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** PunishmentRandomSettings orderStep. */
        orderStep: number;

        /** PunishmentRandomSettings orderSpread. */
        orderSpread: number;

        /** PunishmentRandomSettings maxDifficultyOvershoot. */
        maxDifficultyOvershoot: number;

        /**
         * Creates a new PunishmentRandomSettings instance using the specified properties.
         * @param [properties] Properties to set
         * @returns PunishmentRandomSettings instance
         */
        static create(properties: game.PunishmentRandomSettings.$Shape): game.PunishmentRandomSettings & game.PunishmentRandomSettings.$Shape;
        static create(properties?: game.PunishmentRandomSettings.$Properties): game.PunishmentRandomSettings;

        /**
         * Encodes the specified PunishmentRandomSettings message. Does not implicitly {@link game.PunishmentRandomSettings.verify|verify} messages.
         * @param message PunishmentRandomSettings message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.PunishmentRandomSettings.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified PunishmentRandomSettings message, length delimited. Does not implicitly {@link game.PunishmentRandomSettings.verify|verify} messages.
         * @param message PunishmentRandomSettings message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.PunishmentRandomSettings.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a PunishmentRandomSettings message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.PunishmentRandomSettings & game.PunishmentRandomSettings.$Shape} PunishmentRandomSettings
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.PunishmentRandomSettings & game.PunishmentRandomSettings.$Shape;

        /**
         * Decodes a PunishmentRandomSettings message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.PunishmentRandomSettings & game.PunishmentRandomSettings.$Shape} PunishmentRandomSettings
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.PunishmentRandomSettings & game.PunishmentRandomSettings.$Shape;

        /**
         * Verifies a PunishmentRandomSettings message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a PunishmentRandomSettings message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns PunishmentRandomSettings
         */
        static fromObject(object: { [k: string]: any }): game.PunishmentRandomSettings;

        /**
         * Creates a plain object from a PunishmentRandomSettings message. Also converts values to other types if specified.
         * @param message PunishmentRandomSettings
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.PunishmentRandomSettings, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this PunishmentRandomSettings to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for PunishmentRandomSettings
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace PunishmentRandomSettings {

        /** Properties of a PunishmentRandomSettings. */
        interface $Properties {

            /** PunishmentRandomSettings orderStep */
            orderStep?: (number|null);

            /** PunishmentRandomSettings orderSpread */
            orderSpread?: (number|null);

            /** PunishmentRandomSettings maxDifficultyOvershoot */
            maxDifficultyOvershoot?: (number|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a PunishmentRandomSettings. */
        type $Shape = game.PunishmentRandomSettings.$Properties;
    }

    /**
     * Properties of a PunishmentSubtaskVariant.
     * @deprecated Use game.PunishmentSubtaskVariant.$Properties instead.
     */
    interface IPunishmentSubtaskVariant extends game.PunishmentSubtaskVariant.$Properties {
    }

    /** Represents a PunishmentSubtaskVariant. */
    class PunishmentSubtaskVariant {

        /**
         * Constructs a new PunishmentSubtaskVariant.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.PunishmentSubtaskVariant.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** PunishmentSubtaskVariant factionIds. */
        factionIds: string[];

        /** PunishmentSubtaskVariant text. */
        text: string;

        /** PunishmentSubtaskVariant backgroundImages. */
        backgroundImages: string[];

        /** PunishmentSubtaskVariant backgroundOpacity. */
        backgroundOpacity: number;

        /**
         * Creates a new PunishmentSubtaskVariant instance using the specified properties.
         * @param [properties] Properties to set
         * @returns PunishmentSubtaskVariant instance
         */
        static create(properties: game.PunishmentSubtaskVariant.$Shape): game.PunishmentSubtaskVariant & game.PunishmentSubtaskVariant.$Shape;
        static create(properties?: game.PunishmentSubtaskVariant.$Properties): game.PunishmentSubtaskVariant;

        /**
         * Encodes the specified PunishmentSubtaskVariant message. Does not implicitly {@link game.PunishmentSubtaskVariant.verify|verify} messages.
         * @param message PunishmentSubtaskVariant message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.PunishmentSubtaskVariant.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified PunishmentSubtaskVariant message, length delimited. Does not implicitly {@link game.PunishmentSubtaskVariant.verify|verify} messages.
         * @param message PunishmentSubtaskVariant message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.PunishmentSubtaskVariant.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a PunishmentSubtaskVariant message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.PunishmentSubtaskVariant & game.PunishmentSubtaskVariant.$Shape} PunishmentSubtaskVariant
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.PunishmentSubtaskVariant & game.PunishmentSubtaskVariant.$Shape;

        /**
         * Decodes a PunishmentSubtaskVariant message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.PunishmentSubtaskVariant & game.PunishmentSubtaskVariant.$Shape} PunishmentSubtaskVariant
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.PunishmentSubtaskVariant & game.PunishmentSubtaskVariant.$Shape;

        /**
         * Verifies a PunishmentSubtaskVariant message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a PunishmentSubtaskVariant message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns PunishmentSubtaskVariant
         */
        static fromObject(object: { [k: string]: any }): game.PunishmentSubtaskVariant;

        /**
         * Creates a plain object from a PunishmentSubtaskVariant message. Also converts values to other types if specified.
         * @param message PunishmentSubtaskVariant
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.PunishmentSubtaskVariant, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this PunishmentSubtaskVariant to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for PunishmentSubtaskVariant
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace PunishmentSubtaskVariant {

        /** Properties of a PunishmentSubtaskVariant. */
        interface $Properties {

            /** PunishmentSubtaskVariant factionIds */
            factionIds?: (string[]|null);

            /** PunishmentSubtaskVariant text */
            text?: (string|null);

            /** PunishmentSubtaskVariant backgroundImages */
            backgroundImages?: (string[]|null);

            /** PunishmentSubtaskVariant backgroundOpacity */
            backgroundOpacity?: (number|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a PunishmentSubtaskVariant. */
        type $Shape = game.PunishmentSubtaskVariant.$Properties;
    }

    /**
     * Properties of a PunishmentSubtask.
     * @deprecated Use game.PunishmentSubtask.$Properties instead.
     */
    interface IPunishmentSubtask extends game.PunishmentSubtask.$Properties {
    }

    /** Represents a PunishmentSubtask. */
    class PunishmentSubtask {

        /**
         * Constructs a new PunishmentSubtask.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.PunishmentSubtask.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** PunishmentSubtask variants. */
        variants: game.PunishmentSubtaskVariant.$Properties[];

        /**
         * Creates a new PunishmentSubtask instance using the specified properties.
         * @param [properties] Properties to set
         * @returns PunishmentSubtask instance
         */
        static create(properties: game.PunishmentSubtask.$Shape): game.PunishmentSubtask & game.PunishmentSubtask.$Shape;
        static create(properties?: game.PunishmentSubtask.$Properties): game.PunishmentSubtask;

        /**
         * Encodes the specified PunishmentSubtask message. Does not implicitly {@link game.PunishmentSubtask.verify|verify} messages.
         * @param message PunishmentSubtask message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.PunishmentSubtask.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified PunishmentSubtask message, length delimited. Does not implicitly {@link game.PunishmentSubtask.verify|verify} messages.
         * @param message PunishmentSubtask message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.PunishmentSubtask.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a PunishmentSubtask message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.PunishmentSubtask & game.PunishmentSubtask.$Shape} PunishmentSubtask
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.PunishmentSubtask & game.PunishmentSubtask.$Shape;

        /**
         * Decodes a PunishmentSubtask message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.PunishmentSubtask & game.PunishmentSubtask.$Shape} PunishmentSubtask
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.PunishmentSubtask & game.PunishmentSubtask.$Shape;

        /**
         * Verifies a PunishmentSubtask message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a PunishmentSubtask message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns PunishmentSubtask
         */
        static fromObject(object: { [k: string]: any }): game.PunishmentSubtask;

        /**
         * Creates a plain object from a PunishmentSubtask message. Also converts values to other types if specified.
         * @param message PunishmentSubtask
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.PunishmentSubtask, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this PunishmentSubtask to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for PunishmentSubtask
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace PunishmentSubtask {

        /** Properties of a PunishmentSubtask. */
        interface $Properties {

            /** PunishmentSubtask variants */
            variants?: (game.PunishmentSubtaskVariant.$Properties[]|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a PunishmentSubtask. */
        type $Shape = game.PunishmentSubtask.$Properties;
    }

    /**
     * Properties of a PunishmentSeriesTaskConfig.
     * @deprecated Use game.PunishmentSeriesTaskConfig.$Properties instead.
     */
    interface IPunishmentSeriesTaskConfig extends game.PunishmentSeriesTaskConfig.$Properties {
    }

    /** Represents a PunishmentSeriesTaskConfig. */
    class PunishmentSeriesTaskConfig {

        /**
         * Constructs a new PunishmentSeriesTaskConfig.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.PunishmentSeriesTaskConfig.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** PunishmentSeriesTaskConfig id. */
        id: string;

        /** PunishmentSeriesTaskConfig name. */
        name: string;

        /** PunishmentSeriesTaskConfig roomNamePool. */
        roomNamePool?: (game.RoomNamePool.$Properties|null);

        /** PunishmentSeriesTaskConfig roomBackgroundImages. */
        roomBackgroundImages: string[];

        /** PunishmentSeriesTaskConfig subtasks. */
        subtasks: game.PunishmentSubtask.$Properties[];

        /**
         * Creates a new PunishmentSeriesTaskConfig instance using the specified properties.
         * @param [properties] Properties to set
         * @returns PunishmentSeriesTaskConfig instance
         */
        static create(properties: game.PunishmentSeriesTaskConfig.$Shape): game.PunishmentSeriesTaskConfig & game.PunishmentSeriesTaskConfig.$Shape;
        static create(properties?: game.PunishmentSeriesTaskConfig.$Properties): game.PunishmentSeriesTaskConfig;

        /**
         * Encodes the specified PunishmentSeriesTaskConfig message. Does not implicitly {@link game.PunishmentSeriesTaskConfig.verify|verify} messages.
         * @param message PunishmentSeriesTaskConfig message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.PunishmentSeriesTaskConfig.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified PunishmentSeriesTaskConfig message, length delimited. Does not implicitly {@link game.PunishmentSeriesTaskConfig.verify|verify} messages.
         * @param message PunishmentSeriesTaskConfig message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.PunishmentSeriesTaskConfig.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a PunishmentSeriesTaskConfig message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.PunishmentSeriesTaskConfig & game.PunishmentSeriesTaskConfig.$Shape} PunishmentSeriesTaskConfig
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.PunishmentSeriesTaskConfig & game.PunishmentSeriesTaskConfig.$Shape;

        /**
         * Decodes a PunishmentSeriesTaskConfig message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.PunishmentSeriesTaskConfig & game.PunishmentSeriesTaskConfig.$Shape} PunishmentSeriesTaskConfig
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.PunishmentSeriesTaskConfig & game.PunishmentSeriesTaskConfig.$Shape;

        /**
         * Verifies a PunishmentSeriesTaskConfig message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a PunishmentSeriesTaskConfig message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns PunishmentSeriesTaskConfig
         */
        static fromObject(object: { [k: string]: any }): game.PunishmentSeriesTaskConfig;

        /**
         * Creates a plain object from a PunishmentSeriesTaskConfig message. Also converts values to other types if specified.
         * @param message PunishmentSeriesTaskConfig
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.PunishmentSeriesTaskConfig, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this PunishmentSeriesTaskConfig to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for PunishmentSeriesTaskConfig
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace PunishmentSeriesTaskConfig {

        /** Properties of a PunishmentSeriesTaskConfig. */
        interface $Properties {

            /** PunishmentSeriesTaskConfig id */
            id?: (string|null);

            /** PunishmentSeriesTaskConfig name */
            name?: (string|null);

            /** PunishmentSeriesTaskConfig roomNamePool */
            roomNamePool?: (game.RoomNamePool.$Properties|null);

            /** PunishmentSeriesTaskConfig roomBackgroundImages */
            roomBackgroundImages?: (string[]|null);

            /** PunishmentSeriesTaskConfig subtasks */
            subtasks?: (game.PunishmentSubtask.$Properties[]|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a PunishmentSeriesTaskConfig. */
        type $Shape = game.PunishmentSeriesTaskConfig.$Properties;
    }

    /**
     * Properties of a PunishmentSeriesSummary.
     * @deprecated Use game.PunishmentSeriesSummary.$Properties instead.
     */
    interface IPunishmentSeriesSummary extends game.PunishmentSeriesSummary.$Properties {
    }

    /** Represents a PunishmentSeriesSummary. */
    class PunishmentSeriesSummary {

        /**
         * Constructs a new PunishmentSeriesSummary.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.PunishmentSeriesSummary.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** PunishmentSeriesSummary id. */
        id: string;

        /** PunishmentSeriesSummary name. */
        name: string;

        /** PunishmentSeriesSummary roomNamePool. */
        roomNamePool?: (game.RoomNamePool.$Properties|null);

        /** PunishmentSeriesSummary roomBackgroundImages. */
        roomBackgroundImages: string[];

        /** PunishmentSeriesSummary stepCount. */
        stepCount: number;

        /**
         * Creates a new PunishmentSeriesSummary instance using the specified properties.
         * @param [properties] Properties to set
         * @returns PunishmentSeriesSummary instance
         */
        static create(properties: game.PunishmentSeriesSummary.$Shape): game.PunishmentSeriesSummary & game.PunishmentSeriesSummary.$Shape;
        static create(properties?: game.PunishmentSeriesSummary.$Properties): game.PunishmentSeriesSummary;

        /**
         * Encodes the specified PunishmentSeriesSummary message. Does not implicitly {@link game.PunishmentSeriesSummary.verify|verify} messages.
         * @param message PunishmentSeriesSummary message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.PunishmentSeriesSummary.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified PunishmentSeriesSummary message, length delimited. Does not implicitly {@link game.PunishmentSeriesSummary.verify|verify} messages.
         * @param message PunishmentSeriesSummary message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.PunishmentSeriesSummary.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a PunishmentSeriesSummary message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.PunishmentSeriesSummary & game.PunishmentSeriesSummary.$Shape} PunishmentSeriesSummary
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.PunishmentSeriesSummary & game.PunishmentSeriesSummary.$Shape;

        /**
         * Decodes a PunishmentSeriesSummary message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.PunishmentSeriesSummary & game.PunishmentSeriesSummary.$Shape} PunishmentSeriesSummary
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.PunishmentSeriesSummary & game.PunishmentSeriesSummary.$Shape;

        /**
         * Verifies a PunishmentSeriesSummary message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a PunishmentSeriesSummary message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns PunishmentSeriesSummary
         */
        static fromObject(object: { [k: string]: any }): game.PunishmentSeriesSummary;

        /**
         * Creates a plain object from a PunishmentSeriesSummary message. Also converts values to other types if specified.
         * @param message PunishmentSeriesSummary
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.PunishmentSeriesSummary, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this PunishmentSeriesSummary to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for PunishmentSeriesSummary
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace PunishmentSeriesSummary {

        /** Properties of a PunishmentSeriesSummary. */
        interface $Properties {

            /** PunishmentSeriesSummary id */
            id?: (string|null);

            /** PunishmentSeriesSummary name */
            name?: (string|null);

            /** PunishmentSeriesSummary roomNamePool */
            roomNamePool?: (game.RoomNamePool.$Properties|null);

            /** PunishmentSeriesSummary roomBackgroundImages */
            roomBackgroundImages?: (string[]|null);

            /** PunishmentSeriesSummary stepCount */
            stepCount?: (number|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a PunishmentSeriesSummary. */
        type $Shape = game.PunishmentSeriesSummary.$Properties;
    }

    /**
     * Properties of a GameConfig.
     * @deprecated Use game.GameConfig.$Properties instead.
     */
    interface IGameConfig extends game.GameConfig.$Properties {
    }

    /** Represents a GameConfig. */
    class GameConfig {

        /**
         * Constructs a new GameConfig.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.GameConfig.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** GameConfig id. */
        id: string;

        /** GameConfig name. */
        name: string;

        /** GameConfig description. */
        description: string;

        /**
         * Creates a new GameConfig instance using the specified properties.
         * @param [properties] Properties to set
         * @returns GameConfig instance
         */
        static create(properties: game.GameConfig.$Shape): game.GameConfig & game.GameConfig.$Shape;
        static create(properties?: game.GameConfig.$Properties): game.GameConfig;

        /**
         * Encodes the specified GameConfig message. Does not implicitly {@link game.GameConfig.verify|verify} messages.
         * @param message GameConfig message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.GameConfig.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified GameConfig message, length delimited. Does not implicitly {@link game.GameConfig.verify|verify} messages.
         * @param message GameConfig message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.GameConfig.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a GameConfig message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.GameConfig & game.GameConfig.$Shape} GameConfig
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.GameConfig & game.GameConfig.$Shape;

        /**
         * Decodes a GameConfig message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.GameConfig & game.GameConfig.$Shape} GameConfig
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.GameConfig & game.GameConfig.$Shape;

        /**
         * Verifies a GameConfig message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a GameConfig message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns GameConfig
         */
        static fromObject(object: { [k: string]: any }): game.GameConfig;

        /**
         * Creates a plain object from a GameConfig message. Also converts values to other types if specified.
         * @param message GameConfig
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.GameConfig, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this GameConfig to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for GameConfig
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace GameConfig {

        /** Properties of a GameConfig. */
        interface $Properties {

            /** GameConfig id */
            id?: (string|null);

            /** GameConfig name */
            name?: (string|null);

            /** GameConfig description */
            description?: (string|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a GameConfig. */
        type $Shape = game.GameConfig.$Properties;
    }

    /**
     * Properties of an AnnouncementBoard.
     * @deprecated Use game.AnnouncementBoard.$Properties instead.
     */
    interface IAnnouncementBoard extends game.AnnouncementBoard.$Properties {
    }

    /** Represents an AnnouncementBoard. */
    class AnnouncementBoard {

        /**
         * Constructs a new AnnouncementBoard.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.AnnouncementBoard.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** AnnouncementBoard enabled. */
        enabled: boolean;

        /** AnnouncementBoard title. */
        title: string;

        /** AnnouncementBoard content. */
        content: string;

        /**
         * Creates a new AnnouncementBoard instance using the specified properties.
         * @param [properties] Properties to set
         * @returns AnnouncementBoard instance
         */
        static create(properties: game.AnnouncementBoard.$Shape): game.AnnouncementBoard & game.AnnouncementBoard.$Shape;
        static create(properties?: game.AnnouncementBoard.$Properties): game.AnnouncementBoard;

        /**
         * Encodes the specified AnnouncementBoard message. Does not implicitly {@link game.AnnouncementBoard.verify|verify} messages.
         * @param message AnnouncementBoard message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.AnnouncementBoard.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified AnnouncementBoard message, length delimited. Does not implicitly {@link game.AnnouncementBoard.verify|verify} messages.
         * @param message AnnouncementBoard message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.AnnouncementBoard.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes an AnnouncementBoard message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.AnnouncementBoard & game.AnnouncementBoard.$Shape} AnnouncementBoard
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.AnnouncementBoard & game.AnnouncementBoard.$Shape;

        /**
         * Decodes an AnnouncementBoard message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.AnnouncementBoard & game.AnnouncementBoard.$Shape} AnnouncementBoard
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.AnnouncementBoard & game.AnnouncementBoard.$Shape;

        /**
         * Verifies an AnnouncementBoard message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates an AnnouncementBoard message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns AnnouncementBoard
         */
        static fromObject(object: { [k: string]: any }): game.AnnouncementBoard;

        /**
         * Creates a plain object from an AnnouncementBoard message. Also converts values to other types if specified.
         * @param message AnnouncementBoard
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.AnnouncementBoard, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this AnnouncementBoard to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for AnnouncementBoard
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace AnnouncementBoard {

        /** Properties of an AnnouncementBoard. */
        interface $Properties {

            /** AnnouncementBoard enabled */
            enabled?: (boolean|null);

            /** AnnouncementBoard title */
            title?: (string|null);

            /** AnnouncementBoard content */
            content?: (string|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of an AnnouncementBoard. */
        type $Shape = game.AnnouncementBoard.$Properties;
    }

    /**
     * Properties of a SecurityDisclaimerConfig.
     * @deprecated Use game.SecurityDisclaimerConfig.$Properties instead.
     */
    interface ISecurityDisclaimerConfig extends game.SecurityDisclaimerConfig.$Properties {
    }

    /** Represents a SecurityDisclaimerConfig. */
    class SecurityDisclaimerConfig {

        /**
         * Constructs a new SecurityDisclaimerConfig.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.SecurityDisclaimerConfig.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** SecurityDisclaimerConfig enabled. */
        enabled: boolean;

        /**
         * Creates a new SecurityDisclaimerConfig instance using the specified properties.
         * @param [properties] Properties to set
         * @returns SecurityDisclaimerConfig instance
         */
        static create(properties: game.SecurityDisclaimerConfig.$Shape): game.SecurityDisclaimerConfig & game.SecurityDisclaimerConfig.$Shape;
        static create(properties?: game.SecurityDisclaimerConfig.$Properties): game.SecurityDisclaimerConfig;

        /**
         * Encodes the specified SecurityDisclaimerConfig message. Does not implicitly {@link game.SecurityDisclaimerConfig.verify|verify} messages.
         * @param message SecurityDisclaimerConfig message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.SecurityDisclaimerConfig.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified SecurityDisclaimerConfig message, length delimited. Does not implicitly {@link game.SecurityDisclaimerConfig.verify|verify} messages.
         * @param message SecurityDisclaimerConfig message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.SecurityDisclaimerConfig.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a SecurityDisclaimerConfig message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.SecurityDisclaimerConfig & game.SecurityDisclaimerConfig.$Shape} SecurityDisclaimerConfig
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.SecurityDisclaimerConfig & game.SecurityDisclaimerConfig.$Shape;

        /**
         * Decodes a SecurityDisclaimerConfig message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.SecurityDisclaimerConfig & game.SecurityDisclaimerConfig.$Shape} SecurityDisclaimerConfig
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.SecurityDisclaimerConfig & game.SecurityDisclaimerConfig.$Shape;

        /**
         * Verifies a SecurityDisclaimerConfig message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a SecurityDisclaimerConfig message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns SecurityDisclaimerConfig
         */
        static fromObject(object: { [k: string]: any }): game.SecurityDisclaimerConfig;

        /**
         * Creates a plain object from a SecurityDisclaimerConfig message. Also converts values to other types if specified.
         * @param message SecurityDisclaimerConfig
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.SecurityDisclaimerConfig, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this SecurityDisclaimerConfig to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for SecurityDisclaimerConfig
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace SecurityDisclaimerConfig {

        /** Properties of a SecurityDisclaimerConfig. */
        interface $Properties {

            /** SecurityDisclaimerConfig enabled */
            enabled?: (boolean|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a SecurityDisclaimerConfig. */
        type $Shape = game.SecurityDisclaimerConfig.$Properties;
    }

    /**
     * Properties of a DoublePair.
     * @deprecated Use game.DoublePair.$Properties instead.
     */
    interface IDoublePair extends game.DoublePair.$Properties {
    }

    /** Represents a DoublePair. */
    class DoublePair {

        /**
         * Constructs a new DoublePair.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.DoublePair.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** DoublePair key. */
        key: string;

        /** DoublePair value. */
        value: number;

        /**
         * Creates a new DoublePair instance using the specified properties.
         * @param [properties] Properties to set
         * @returns DoublePair instance
         */
        static create(properties: game.DoublePair.$Shape): game.DoublePair & game.DoublePair.$Shape;
        static create(properties?: game.DoublePair.$Properties): game.DoublePair;

        /**
         * Encodes the specified DoublePair message. Does not implicitly {@link game.DoublePair.verify|verify} messages.
         * @param message DoublePair message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.DoublePair.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified DoublePair message, length delimited. Does not implicitly {@link game.DoublePair.verify|verify} messages.
         * @param message DoublePair message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.DoublePair.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a DoublePair message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.DoublePair & game.DoublePair.$Shape} DoublePair
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.DoublePair & game.DoublePair.$Shape;

        /**
         * Decodes a DoublePair message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.DoublePair & game.DoublePair.$Shape} DoublePair
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.DoublePair & game.DoublePair.$Shape;

        /**
         * Verifies a DoublePair message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a DoublePair message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns DoublePair
         */
        static fromObject(object: { [k: string]: any }): game.DoublePair;

        /**
         * Creates a plain object from a DoublePair message. Also converts values to other types if specified.
         * @param message DoublePair
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.DoublePair, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this DoublePair to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for DoublePair
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace DoublePair {

        /** Properties of a DoublePair. */
        interface $Properties {

            /** DoublePair key */
            key?: (string|null);

            /** DoublePair value */
            value?: (number|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a DoublePair. */
        type $Shape = game.DoublePair.$Properties;
    }

    /**
     * Properties of an ExtremeModeConfig.
     * @deprecated Use game.ExtremeModeConfig.$Properties instead.
     */
    interface IExtremeModeConfig extends game.ExtremeModeConfig.$Properties {
    }

    /** Represents an ExtremeModeConfig. */
    class ExtremeModeConfig {

        /**
         * Constructs a new ExtremeModeConfig.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.ExtremeModeConfig.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** ExtremeModeConfig label. */
        label: string;

        /** ExtremeModeConfig emoji. */
        emoji: string;

        /** ExtremeModeConfig cooldownHours. */
        cooldownHours: number;

        /** ExtremeModeConfig positiveLossRates. */
        positiveLossRates: game.DoublePair.$Properties[];

        /** ExtremeModeConfig negativeWinRates. */
        negativeWinRates: game.DoublePair.$Properties[];

        /** ExtremeModeConfig hourlyDecay. */
        hourlyDecay: game.DoublePair.$Properties[];

        /** ExtremeModeConfig winStreakThreshold. */
        winStreakThreshold: number;

        /** ExtremeModeConfig winStreakCrashChance. */
        winStreakCrashChance: number;

        /** ExtremeModeConfig crashTargetPoints. */
        crashTargetPoints: number;

        /** ExtremeModeConfig forceCloseWarning. */
        forceCloseWarning: string;

        /** ExtremeModeConfig forceRenameMinPoints. */
        forceRenameMinPoints: number;

        /** ExtremeModeConfig forceRenameProtectHours. */
        forceRenameProtectHours: number;

        /**
         * Creates a new ExtremeModeConfig instance using the specified properties.
         * @param [properties] Properties to set
         * @returns ExtremeModeConfig instance
         */
        static create(properties: game.ExtremeModeConfig.$Shape): game.ExtremeModeConfig & game.ExtremeModeConfig.$Shape;
        static create(properties?: game.ExtremeModeConfig.$Properties): game.ExtremeModeConfig;

        /**
         * Encodes the specified ExtremeModeConfig message. Does not implicitly {@link game.ExtremeModeConfig.verify|verify} messages.
         * @param message ExtremeModeConfig message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.ExtremeModeConfig.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified ExtremeModeConfig message, length delimited. Does not implicitly {@link game.ExtremeModeConfig.verify|verify} messages.
         * @param message ExtremeModeConfig message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.ExtremeModeConfig.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes an ExtremeModeConfig message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.ExtremeModeConfig & game.ExtremeModeConfig.$Shape} ExtremeModeConfig
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.ExtremeModeConfig & game.ExtremeModeConfig.$Shape;

        /**
         * Decodes an ExtremeModeConfig message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.ExtremeModeConfig & game.ExtremeModeConfig.$Shape} ExtremeModeConfig
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.ExtremeModeConfig & game.ExtremeModeConfig.$Shape;

        /**
         * Verifies an ExtremeModeConfig message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates an ExtremeModeConfig message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns ExtremeModeConfig
         */
        static fromObject(object: { [k: string]: any }): game.ExtremeModeConfig;

        /**
         * Creates a plain object from an ExtremeModeConfig message. Also converts values to other types if specified.
         * @param message ExtremeModeConfig
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.ExtremeModeConfig, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this ExtremeModeConfig to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for ExtremeModeConfig
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace ExtremeModeConfig {

        /** Properties of an ExtremeModeConfig. */
        interface $Properties {

            /** ExtremeModeConfig label */
            label?: (string|null);

            /** ExtremeModeConfig emoji */
            emoji?: (string|null);

            /** ExtremeModeConfig cooldownHours */
            cooldownHours?: (number|null);

            /** ExtremeModeConfig positiveLossRates */
            positiveLossRates?: (game.DoublePair.$Properties[]|null);

            /** ExtremeModeConfig negativeWinRates */
            negativeWinRates?: (game.DoublePair.$Properties[]|null);

            /** ExtremeModeConfig hourlyDecay */
            hourlyDecay?: (game.DoublePair.$Properties[]|null);

            /** ExtremeModeConfig winStreakThreshold */
            winStreakThreshold?: (number|null);

            /** ExtremeModeConfig winStreakCrashChance */
            winStreakCrashChance?: (number|null);

            /** ExtremeModeConfig crashTargetPoints */
            crashTargetPoints?: (number|null);

            /** ExtremeModeConfig forceCloseWarning */
            forceCloseWarning?: (string|null);

            /** ExtremeModeConfig forceRenameMinPoints */
            forceRenameMinPoints?: (number|null);

            /** ExtremeModeConfig forceRenameProtectHours */
            forceRenameProtectHours?: (number|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of an ExtremeModeConfig. */
        type $Shape = game.ExtremeModeConfig.$Properties;
    }

    /**
     * Properties of a SiteConfig.
     * @deprecated Use game.SiteConfig.$Properties instead.
     */
    interface ISiteConfig extends game.SiteConfig.$Properties {
    }

    /** Represents a SiteConfig. */
    class SiteConfig {

        /**
         * Constructs a new SiteConfig.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.SiteConfig.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** SiteConfig name. */
        name: string;

        /** SiteConfig description. */
        description: string;

        /** SiteConfig adminPassword. */
        adminPassword: string;

        /**
         * Creates a new SiteConfig instance using the specified properties.
         * @param [properties] Properties to set
         * @returns SiteConfig instance
         */
        static create(properties: game.SiteConfig.$Shape): game.SiteConfig & game.SiteConfig.$Shape;
        static create(properties?: game.SiteConfig.$Properties): game.SiteConfig;

        /**
         * Encodes the specified SiteConfig message. Does not implicitly {@link game.SiteConfig.verify|verify} messages.
         * @param message SiteConfig message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.SiteConfig.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified SiteConfig message, length delimited. Does not implicitly {@link game.SiteConfig.verify|verify} messages.
         * @param message SiteConfig message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.SiteConfig.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a SiteConfig message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.SiteConfig & game.SiteConfig.$Shape} SiteConfig
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.SiteConfig & game.SiteConfig.$Shape;

        /**
         * Decodes a SiteConfig message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.SiteConfig & game.SiteConfig.$Shape} SiteConfig
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.SiteConfig & game.SiteConfig.$Shape;

        /**
         * Verifies a SiteConfig message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a SiteConfig message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns SiteConfig
         */
        static fromObject(object: { [k: string]: any }): game.SiteConfig;

        /**
         * Creates a plain object from a SiteConfig message. Also converts values to other types if specified.
         * @param message SiteConfig
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.SiteConfig, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this SiteConfig to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for SiteConfig
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace SiteConfig {

        /** Properties of a SiteConfig. */
        interface $Properties {

            /** SiteConfig name */
            name?: (string|null);

            /** SiteConfig description */
            description?: (string|null);

            /** SiteConfig adminPassword */
            adminPassword?: (string|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a SiteConfig. */
        type $Shape = game.SiteConfig.$Properties;
    }

    /**
     * Properties of an AccessControlConfig.
     * @deprecated Use game.AccessControlConfig.$Properties instead.
     */
    interface IAccessControlConfig extends game.AccessControlConfig.$Properties {
    }

    /** Represents an AccessControlConfig. */
    class AccessControlConfig {

        /**
         * Constructs a new AccessControlConfig.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.AccessControlConfig.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** AccessControlConfig maxOnlinePerIp. */
        maxOnlinePerIp: number;

        /** AccessControlConfig maxCreatesPer_10Min. */
        maxCreatesPer_10Min: number;

        /** AccessControlConfig ipBackstopMultiplier. */
        ipBackstopMultiplier: number;

        /** AccessControlConfig ipBackstopMinLimit. */
        ipBackstopMinLimit: number;

        /** AccessControlConfig maxSessionIssuePerIp. */
        maxSessionIssuePerIp: number;

        /** AccessControlConfig maxOnlinePerIpTotal. */
        maxOnlinePerIpTotal: number;

        /** AccessControlConfig maxCreatesPerIp. */
        maxCreatesPerIp: number;

        /** AccessControlConfig maxActiveRoomsPerOwner. */
        maxActiveRoomsPerOwner: number;

        /** AccessControlConfig maxProofUploadsPerPlayer. */
        maxProofUploadsPerPlayer: number;

        /** AccessControlConfig registrationDisabled. */
        registrationDisabled: boolean;

        /**
         * Creates a new AccessControlConfig instance using the specified properties.
         * @param [properties] Properties to set
         * @returns AccessControlConfig instance
         */
        static create(properties: game.AccessControlConfig.$Shape): game.AccessControlConfig & game.AccessControlConfig.$Shape;
        static create(properties?: game.AccessControlConfig.$Properties): game.AccessControlConfig;

        /**
         * Encodes the specified AccessControlConfig message. Does not implicitly {@link game.AccessControlConfig.verify|verify} messages.
         * @param message AccessControlConfig message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.AccessControlConfig.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified AccessControlConfig message, length delimited. Does not implicitly {@link game.AccessControlConfig.verify|verify} messages.
         * @param message AccessControlConfig message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.AccessControlConfig.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes an AccessControlConfig message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.AccessControlConfig & game.AccessControlConfig.$Shape} AccessControlConfig
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.AccessControlConfig & game.AccessControlConfig.$Shape;

        /**
         * Decodes an AccessControlConfig message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.AccessControlConfig & game.AccessControlConfig.$Shape} AccessControlConfig
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.AccessControlConfig & game.AccessControlConfig.$Shape;

        /**
         * Verifies an AccessControlConfig message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates an AccessControlConfig message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns AccessControlConfig
         */
        static fromObject(object: { [k: string]: any }): game.AccessControlConfig;

        /**
         * Creates a plain object from an AccessControlConfig message. Also converts values to other types if specified.
         * @param message AccessControlConfig
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.AccessControlConfig, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this AccessControlConfig to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for AccessControlConfig
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace AccessControlConfig {

        /** Properties of an AccessControlConfig. */
        interface $Properties {

            /** AccessControlConfig maxOnlinePerIp */
            maxOnlinePerIp?: (number|null);

            /** AccessControlConfig maxCreatesPer_10Min */
            maxCreatesPer_10Min?: (number|null);

            /** AccessControlConfig ipBackstopMultiplier */
            ipBackstopMultiplier?: (number|null);

            /** AccessControlConfig ipBackstopMinLimit */
            ipBackstopMinLimit?: (number|null);

            /** AccessControlConfig maxSessionIssuePerIp */
            maxSessionIssuePerIp?: (number|null);

            /** AccessControlConfig maxOnlinePerIpTotal */
            maxOnlinePerIpTotal?: (number|null);

            /** AccessControlConfig maxCreatesPerIp */
            maxCreatesPerIp?: (number|null);

            /** AccessControlConfig maxActiveRoomsPerOwner */
            maxActiveRoomsPerOwner?: (number|null);

            /** AccessControlConfig maxProofUploadsPerPlayer */
            maxProofUploadsPerPlayer?: (number|null);

            /** AccessControlConfig registrationDisabled */
            registrationDisabled?: (boolean|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of an AccessControlConfig. */
        type $Shape = game.AccessControlConfig.$Properties;
    }

    /**
     * Properties of a NameWarConfig.
     * @deprecated Use game.NameWarConfig.$Properties instead.
     */
    interface INameWarConfig extends game.NameWarConfig.$Properties {
    }

    /** Represents a NameWarConfig. */
    class NameWarConfig {

        /**
         * Constructs a new NameWarConfig.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.NameWarConfig.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** NameWarConfig penaltyPrefix. */
        penaltyPrefix: string;

        /** NameWarConfig loserPanelTitle. */
        loserPanelTitle: string;

        /** NameWarConfig escapeTitle. */
        escapeTitle: string;

        /** NameWarConfig renamePanelTitle. */
        renamePanelTitle: string;

        /** NameWarConfig nameWarLoserLabel. */
        nameWarLoserLabel: string;

        /** NameWarConfig extremeForceClosedLabel. */
        extremeForceClosedLabel: string;

        /** NameWarConfig penaltyThreshold. */
        penaltyThreshold: number;

        /** NameWarConfig renameMinPoints. */
        renameMinPoints: number;

        /**
         * Creates a new NameWarConfig instance using the specified properties.
         * @param [properties] Properties to set
         * @returns NameWarConfig instance
         */
        static create(properties: game.NameWarConfig.$Shape): game.NameWarConfig & game.NameWarConfig.$Shape;
        static create(properties?: game.NameWarConfig.$Properties): game.NameWarConfig;

        /**
         * Encodes the specified NameWarConfig message. Does not implicitly {@link game.NameWarConfig.verify|verify} messages.
         * @param message NameWarConfig message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.NameWarConfig.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified NameWarConfig message, length delimited. Does not implicitly {@link game.NameWarConfig.verify|verify} messages.
         * @param message NameWarConfig message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.NameWarConfig.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a NameWarConfig message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.NameWarConfig & game.NameWarConfig.$Shape} NameWarConfig
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.NameWarConfig & game.NameWarConfig.$Shape;

        /**
         * Decodes a NameWarConfig message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.NameWarConfig & game.NameWarConfig.$Shape} NameWarConfig
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.NameWarConfig & game.NameWarConfig.$Shape;

        /**
         * Verifies a NameWarConfig message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a NameWarConfig message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns NameWarConfig
         */
        static fromObject(object: { [k: string]: any }): game.NameWarConfig;

        /**
         * Creates a plain object from a NameWarConfig message. Also converts values to other types if specified.
         * @param message NameWarConfig
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.NameWarConfig, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this NameWarConfig to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for NameWarConfig
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace NameWarConfig {

        /** Properties of a NameWarConfig. */
        interface $Properties {

            /** NameWarConfig penaltyPrefix */
            penaltyPrefix?: (string|null);

            /** NameWarConfig loserPanelTitle */
            loserPanelTitle?: (string|null);

            /** NameWarConfig escapeTitle */
            escapeTitle?: (string|null);

            /** NameWarConfig renamePanelTitle */
            renamePanelTitle?: (string|null);

            /** NameWarConfig nameWarLoserLabel */
            nameWarLoserLabel?: (string|null);

            /** NameWarConfig extremeForceClosedLabel */
            extremeForceClosedLabel?: (string|null);

            /** NameWarConfig penaltyThreshold */
            penaltyThreshold?: (number|null);

            /** NameWarConfig renameMinPoints */
            renameMinPoints?: (number|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a NameWarConfig. */
        type $Shape = game.NameWarConfig.$Properties;
    }

    /**
     * Properties of a GiveawayConfig.
     * @deprecated Use game.GiveawayConfig.$Properties instead.
     */
    interface IGiveawayConfig extends game.GiveawayConfig.$Properties {
    }

    /** Represents a GiveawayConfig. */
    class GiveawayConfig {

        /**
         * Constructs a new GiveawayConfig.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.GiveawayConfig.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** GiveawayConfig panelTitle. */
        panelTitle: string;

        /** GiveawayConfig panelDescription. */
        panelDescription: string;

        /** GiveawayConfig submitPlaceholder. */
        submitPlaceholder: string;

        /** GiveawayConfig emptyText. */
        emptyText: string;

        /** GiveawayConfig activeBoostValue. */
        activeBoostValue: number;

        /** GiveawayConfig winPenaltyValue. */
        winPenaltyValue: number;

        /** GiveawayConfig likeVoteLimitPerHour. */
        likeVoteLimitPerHour: number;

        /** GiveawayConfig likeVoteValue. */
        likeVoteValue: number;

        /** GiveawayConfig dislikeVoteLimitPerHour. */
        dislikeVoteLimitPerHour: number;

        /** GiveawayConfig dislikeVoteValue. */
        dislikeVoteValue: number;

        /** GiveawayConfig petLikeVoteLimitPerHour. */
        petLikeVoteLimitPerHour: number;

        /** GiveawayConfig petDislikeVoteLimitPerHour. */
        petDislikeVoteLimitPerHour: number;

        /** GiveawayConfig masterLikeVoteLimitPerHour. */
        masterLikeVoteLimitPerHour: number;

        /** GiveawayConfig masterDislikeVoteLimitPerHour. */
        masterDislikeVoteLimitPerHour: number;

        /** GiveawayConfig petLikeVoteValue. */
        petLikeVoteValue: number;

        /** GiveawayConfig petDislikeVoteValue. */
        petDislikeVoteValue: number;

        /** GiveawayConfig masterLikeVoteValue. */
        masterLikeVoteValue: number;

        /** GiveawayConfig masterDislikeVoteValue. */
        masterDislikeVoteValue: number;

        /**
         * Creates a new GiveawayConfig instance using the specified properties.
         * @param [properties] Properties to set
         * @returns GiveawayConfig instance
         */
        static create(properties: game.GiveawayConfig.$Shape): game.GiveawayConfig & game.GiveawayConfig.$Shape;
        static create(properties?: game.GiveawayConfig.$Properties): game.GiveawayConfig;

        /**
         * Encodes the specified GiveawayConfig message. Does not implicitly {@link game.GiveawayConfig.verify|verify} messages.
         * @param message GiveawayConfig message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.GiveawayConfig.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified GiveawayConfig message, length delimited. Does not implicitly {@link game.GiveawayConfig.verify|verify} messages.
         * @param message GiveawayConfig message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.GiveawayConfig.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a GiveawayConfig message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.GiveawayConfig & game.GiveawayConfig.$Shape} GiveawayConfig
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.GiveawayConfig & game.GiveawayConfig.$Shape;

        /**
         * Decodes a GiveawayConfig message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.GiveawayConfig & game.GiveawayConfig.$Shape} GiveawayConfig
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.GiveawayConfig & game.GiveawayConfig.$Shape;

        /**
         * Verifies a GiveawayConfig message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a GiveawayConfig message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns GiveawayConfig
         */
        static fromObject(object: { [k: string]: any }): game.GiveawayConfig;

        /**
         * Creates a plain object from a GiveawayConfig message. Also converts values to other types if specified.
         * @param message GiveawayConfig
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.GiveawayConfig, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this GiveawayConfig to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for GiveawayConfig
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace GiveawayConfig {

        /** Properties of a GiveawayConfig. */
        interface $Properties {

            /** GiveawayConfig panelTitle */
            panelTitle?: (string|null);

            /** GiveawayConfig panelDescription */
            panelDescription?: (string|null);

            /** GiveawayConfig submitPlaceholder */
            submitPlaceholder?: (string|null);

            /** GiveawayConfig emptyText */
            emptyText?: (string|null);

            /** GiveawayConfig activeBoostValue */
            activeBoostValue?: (number|null);

            /** GiveawayConfig winPenaltyValue */
            winPenaltyValue?: (number|null);

            /** GiveawayConfig likeVoteLimitPerHour */
            likeVoteLimitPerHour?: (number|null);

            /** GiveawayConfig likeVoteValue */
            likeVoteValue?: (number|null);

            /** GiveawayConfig dislikeVoteLimitPerHour */
            dislikeVoteLimitPerHour?: (number|null);

            /** GiveawayConfig dislikeVoteValue */
            dislikeVoteValue?: (number|null);

            /** GiveawayConfig petLikeVoteLimitPerHour */
            petLikeVoteLimitPerHour?: (number|null);

            /** GiveawayConfig petDislikeVoteLimitPerHour */
            petDislikeVoteLimitPerHour?: (number|null);

            /** GiveawayConfig masterLikeVoteLimitPerHour */
            masterLikeVoteLimitPerHour?: (number|null);

            /** GiveawayConfig masterDislikeVoteLimitPerHour */
            masterDislikeVoteLimitPerHour?: (number|null);

            /** GiveawayConfig petLikeVoteValue */
            petLikeVoteValue?: (number|null);

            /** GiveawayConfig petDislikeVoteValue */
            petDislikeVoteValue?: (number|null);

            /** GiveawayConfig masterLikeVoteValue */
            masterLikeVoteValue?: (number|null);

            /** GiveawayConfig masterDislikeVoteValue */
            masterDislikeVoteValue?: (number|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a GiveawayConfig. */
        type $Shape = game.GiveawayConfig.$Properties;
    }

    /**
     * Properties of a PetBondConfig.
     * @deprecated Use game.PetBondConfig.$Properties instead.
     */
    interface IPetBondConfig extends game.PetBondConfig.$Properties {
    }

    /** Represents a PetBondConfig. */
    class PetBondConfig {

        /**
         * Constructs a new PetBondConfig.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.PetBondConfig.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** PetBondConfig panelTitle. */
        panelTitle: string;

        /** PetBondConfig maxPetsPerMaster. */
        maxPetsPerMaster: number;

        /** PetBondConfig maxMastersPerPet. */
        maxMastersPerPet: number;

        /** PetBondConfig maxTitleLength. */
        maxTitleLength: number;

        /**
         * Creates a new PetBondConfig instance using the specified properties.
         * @param [properties] Properties to set
         * @returns PetBondConfig instance
         */
        static create(properties: game.PetBondConfig.$Shape): game.PetBondConfig & game.PetBondConfig.$Shape;
        static create(properties?: game.PetBondConfig.$Properties): game.PetBondConfig;

        /**
         * Encodes the specified PetBondConfig message. Does not implicitly {@link game.PetBondConfig.verify|verify} messages.
         * @param message PetBondConfig message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.PetBondConfig.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified PetBondConfig message, length delimited. Does not implicitly {@link game.PetBondConfig.verify|verify} messages.
         * @param message PetBondConfig message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.PetBondConfig.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a PetBondConfig message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.PetBondConfig & game.PetBondConfig.$Shape} PetBondConfig
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.PetBondConfig & game.PetBondConfig.$Shape;

        /**
         * Decodes a PetBondConfig message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.PetBondConfig & game.PetBondConfig.$Shape} PetBondConfig
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.PetBondConfig & game.PetBondConfig.$Shape;

        /**
         * Verifies a PetBondConfig message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a PetBondConfig message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns PetBondConfig
         */
        static fromObject(object: { [k: string]: any }): game.PetBondConfig;

        /**
         * Creates a plain object from a PetBondConfig message. Also converts values to other types if specified.
         * @param message PetBondConfig
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.PetBondConfig, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this PetBondConfig to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for PetBondConfig
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace PetBondConfig {

        /** Properties of a PetBondConfig. */
        interface $Properties {

            /** PetBondConfig panelTitle */
            panelTitle?: (string|null);

            /** PetBondConfig maxPetsPerMaster */
            maxPetsPerMaster?: (number|null);

            /** PetBondConfig maxMastersPerPet */
            maxMastersPerPet?: (number|null);

            /** PetBondConfig maxTitleLength */
            maxTitleLength?: (number|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a PetBondConfig. */
        type $Shape = game.PetBondConfig.$Properties;
    }

    /**
     * Properties of an AppConfig.
     * @deprecated Use game.AppConfig.$Properties instead.
     */
    interface IAppConfig extends game.AppConfig.$Properties {
    }

    /** Represents an AppConfig. */
    class AppConfig {

        /**
         * Constructs a new AppConfig.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.AppConfig.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** AppConfig site. */
        site?: (game.SiteConfig.$Properties|null);

        /** AppConfig announcementBoard. */
        announcementBoard?: (game.AnnouncementBoard.$Properties|null);

        /** AppConfig genders. */
        genders: game.GenderOption.$Properties[];

        /** AppConfig genderFactions. */
        genderFactions: game.GenderFaction.$Properties[];

        /** AppConfig titles. */
        titles: game.TitleSegment.$Properties[];

        /** AppConfig punishments. */
        punishments: game.PunishmentConfig.$Properties[];

        /** AppConfig playerPunishmentRoomNamePool. */
        playerPunishmentRoomNamePool?: (game.RoomNamePool.$Properties|null);

        /** AppConfig roomTags. */
        roomTags: string[];

        /** AppConfig roomInfoTags. */
        roomInfoTags: game.RoomInfoTagEntry.$Properties[];

        /** AppConfig accessControl. */
        accessControl?: (game.AccessControlConfig.$Properties|null);

        /** AppConfig nameWar. */
        nameWar?: (game.NameWarConfig.$Properties|null);

        /** AppConfig giveaway. */
        giveaway?: (game.GiveawayConfig.$Properties|null);

        /** AppConfig extremeMode. */
        extremeMode?: (game.ExtremeModeConfig.$Properties|null);

        /** AppConfig games. */
        games: game.GameConfig.$Properties[];

        /** AppConfig messages. */
        messages: game.StringPair.$Properties[];

        /** AppConfig securityDisclaimer. */
        securityDisclaimer?: (game.SecurityDisclaimerConfig.$Properties|null);

        /** AppConfig rankedScore. */
        rankedScore?: (game.RankedScoreConfig.$Properties|null);

        /** AppConfig petBond. */
        petBond?: (game.PetBondConfig.$Properties|null);

        /** AppConfig titleTagStyles. */
        titleTagStyles: game.TitleTagStyleEntry.$Properties[];

        /** AppConfig punishmentTags. */
        punishmentTags: game.PunishmentTagConfig.$Properties[];

        /** AppConfig punishmentTasks. */
        punishmentTasks: game.PunishmentTaskConfig.$Properties[];

        /** AppConfig punishmentSeriesTasks. */
        punishmentSeriesTasks: game.PunishmentSeriesTaskConfig.$Properties[];

        /** AppConfig punishmentRandomSettings. */
        punishmentRandomSettings?: (game.PunishmentRandomSettings.$Properties|null);

        /** AppConfig punishmentSeriesSummaries. */
        punishmentSeriesSummaries: game.PunishmentSeriesSummary.$Properties[];

        /**
         * Creates a new AppConfig instance using the specified properties.
         * @param [properties] Properties to set
         * @returns AppConfig instance
         */
        static create(properties: game.AppConfig.$Shape): game.AppConfig & game.AppConfig.$Shape;
        static create(properties?: game.AppConfig.$Properties): game.AppConfig;

        /**
         * Encodes the specified AppConfig message. Does not implicitly {@link game.AppConfig.verify|verify} messages.
         * @param message AppConfig message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.AppConfig.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified AppConfig message, length delimited. Does not implicitly {@link game.AppConfig.verify|verify} messages.
         * @param message AppConfig message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.AppConfig.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes an AppConfig message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.AppConfig & game.AppConfig.$Shape} AppConfig
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.AppConfig & game.AppConfig.$Shape;

        /**
         * Decodes an AppConfig message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.AppConfig & game.AppConfig.$Shape} AppConfig
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.AppConfig & game.AppConfig.$Shape;

        /**
         * Verifies an AppConfig message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates an AppConfig message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns AppConfig
         */
        static fromObject(object: { [k: string]: any }): game.AppConfig;

        /**
         * Creates a plain object from an AppConfig message. Also converts values to other types if specified.
         * @param message AppConfig
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.AppConfig, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this AppConfig to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for AppConfig
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace AppConfig {

        /** Properties of an AppConfig. */
        interface $Properties {

            /** AppConfig site */
            site?: (game.SiteConfig.$Properties|null);

            /** AppConfig announcementBoard */
            announcementBoard?: (game.AnnouncementBoard.$Properties|null);

            /** AppConfig genders */
            genders?: (game.GenderOption.$Properties[]|null);

            /** AppConfig genderFactions */
            genderFactions?: (game.GenderFaction.$Properties[]|null);

            /** AppConfig titles */
            titles?: (game.TitleSegment.$Properties[]|null);

            /** AppConfig punishments */
            punishments?: (game.PunishmentConfig.$Properties[]|null);

            /** AppConfig playerPunishmentRoomNamePool */
            playerPunishmentRoomNamePool?: (game.RoomNamePool.$Properties|null);

            /** AppConfig roomTags */
            roomTags?: (string[]|null);

            /** AppConfig roomInfoTags */
            roomInfoTags?: (game.RoomInfoTagEntry.$Properties[]|null);

            /** AppConfig accessControl */
            accessControl?: (game.AccessControlConfig.$Properties|null);

            /** AppConfig nameWar */
            nameWar?: (game.NameWarConfig.$Properties|null);

            /** AppConfig giveaway */
            giveaway?: (game.GiveawayConfig.$Properties|null);

            /** AppConfig extremeMode */
            extremeMode?: (game.ExtremeModeConfig.$Properties|null);

            /** AppConfig games */
            games?: (game.GameConfig.$Properties[]|null);

            /** AppConfig messages */
            messages?: (game.StringPair.$Properties[]|null);

            /** AppConfig securityDisclaimer */
            securityDisclaimer?: (game.SecurityDisclaimerConfig.$Properties|null);

            /** AppConfig rankedScore */
            rankedScore?: (game.RankedScoreConfig.$Properties|null);

            /** AppConfig petBond */
            petBond?: (game.PetBondConfig.$Properties|null);

            /** AppConfig titleTagStyles */
            titleTagStyles?: (game.TitleTagStyleEntry.$Properties[]|null);

            /** AppConfig punishmentTags */
            punishmentTags?: (game.PunishmentTagConfig.$Properties[]|null);

            /** AppConfig punishmentTasks */
            punishmentTasks?: (game.PunishmentTaskConfig.$Properties[]|null);

            /** AppConfig punishmentSeriesTasks */
            punishmentSeriesTasks?: (game.PunishmentSeriesTaskConfig.$Properties[]|null);

            /** AppConfig punishmentRandomSettings */
            punishmentRandomSettings?: (game.PunishmentRandomSettings.$Properties|null);

            /** AppConfig punishmentSeriesSummaries */
            punishmentSeriesSummaries?: (game.PunishmentSeriesSummary.$Properties[]|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of an AppConfig. */
        type $Shape = game.AppConfig.$Properties;
    }

    /**
     * Properties of a RankedScoreConfig.
     * @deprecated Use game.RankedScoreConfig.$Properties instead.
     */
    interface IRankedScoreConfig extends game.RankedScoreConfig.$Properties {
    }

    /** Represents a RankedScoreConfig. */
    class RankedScoreConfig {

        /**
         * Constructs a new RankedScoreConfig.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.RankedScoreConfig.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** RankedScoreConfig max. */
        max: number;

        /** RankedScoreConfig min. */
        min: number;

        /** RankedScoreConfig nameWarMin. */
        nameWarMin: number;

        /** RankedScoreConfig dailyDecayRatio. */
        dailyDecayRatio: number;

        /**
         * Creates a new RankedScoreConfig instance using the specified properties.
         * @param [properties] Properties to set
         * @returns RankedScoreConfig instance
         */
        static create(properties: game.RankedScoreConfig.$Shape): game.RankedScoreConfig & game.RankedScoreConfig.$Shape;
        static create(properties?: game.RankedScoreConfig.$Properties): game.RankedScoreConfig;

        /**
         * Encodes the specified RankedScoreConfig message. Does not implicitly {@link game.RankedScoreConfig.verify|verify} messages.
         * @param message RankedScoreConfig message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.RankedScoreConfig.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified RankedScoreConfig message, length delimited. Does not implicitly {@link game.RankedScoreConfig.verify|verify} messages.
         * @param message RankedScoreConfig message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.RankedScoreConfig.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a RankedScoreConfig message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.RankedScoreConfig & game.RankedScoreConfig.$Shape} RankedScoreConfig
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.RankedScoreConfig & game.RankedScoreConfig.$Shape;

        /**
         * Decodes a RankedScoreConfig message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.RankedScoreConfig & game.RankedScoreConfig.$Shape} RankedScoreConfig
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.RankedScoreConfig & game.RankedScoreConfig.$Shape;

        /**
         * Verifies a RankedScoreConfig message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a RankedScoreConfig message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns RankedScoreConfig
         */
        static fromObject(object: { [k: string]: any }): game.RankedScoreConfig;

        /**
         * Creates a plain object from a RankedScoreConfig message. Also converts values to other types if specified.
         * @param message RankedScoreConfig
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.RankedScoreConfig, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this RankedScoreConfig to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for RankedScoreConfig
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace RankedScoreConfig {

        /** Properties of a RankedScoreConfig. */
        interface $Properties {

            /** RankedScoreConfig max */
            max?: (number|null);

            /** RankedScoreConfig min */
            min?: (number|null);

            /** RankedScoreConfig nameWarMin */
            nameWarMin?: (number|null);

            /** RankedScoreConfig dailyDecayRatio */
            dailyDecayRatio?: (number|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a RankedScoreConfig. */
        type $Shape = game.RankedScoreConfig.$Properties;
    }

    /**
     * Properties of a StateDocument.
     * @deprecated Use game.StateDocument.$Properties instead.
     */
    interface IStateDocument extends game.StateDocument.$Properties {
    }

    /** Represents a StateDocument. */
    class StateDocument {

        /**
         * Constructs a new StateDocument.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.StateDocument.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** StateDocument lobby. */
        lobby?: (game.LobbySnapshot.$Properties|null);

        /** StateDocument room. */
        room?: (game.RoomSnapshot.$Properties|null);

        /** StateDocument config. */
        config?: (game.AppConfig.$Properties|null);

        /** StateDocument doc. */
        doc?: ("lobby"|"room"|"config");

        /**
         * Creates a new StateDocument instance using the specified properties.
         * @param [properties] Properties to set
         * @returns StateDocument instance
         */
        static create(properties: game.StateDocument.$Shape): game.StateDocument & game.StateDocument.$Shape;
        static create(properties?: game.StateDocument.$Properties): game.StateDocument;

        /**
         * Encodes the specified StateDocument message. Does not implicitly {@link game.StateDocument.verify|verify} messages.
         * @param message StateDocument message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.StateDocument.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified StateDocument message, length delimited. Does not implicitly {@link game.StateDocument.verify|verify} messages.
         * @param message StateDocument message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.StateDocument.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a StateDocument message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.StateDocument & game.StateDocument.$Shape} StateDocument
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.StateDocument & game.StateDocument.$Shape;

        /**
         * Decodes a StateDocument message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.StateDocument & game.StateDocument.$Shape} StateDocument
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.StateDocument & game.StateDocument.$Shape;

        /**
         * Verifies a StateDocument message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a StateDocument message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns StateDocument
         */
        static fromObject(object: { [k: string]: any }): game.StateDocument;

        /**
         * Creates a plain object from a StateDocument message. Also converts values to other types if specified.
         * @param message StateDocument
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.StateDocument, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this StateDocument to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for StateDocument
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace StateDocument {

        /** Properties of a StateDocument. */
        interface $Properties {

            /** StateDocument lobby */
            lobby?: (game.LobbySnapshot.$Properties|null);

            /** StateDocument room */
            room?: (game.RoomSnapshot.$Properties|null);

            /** StateDocument config */
            config?: (game.AppConfig.$Properties|null);

            /** StateDocument doc */
            doc?: ("lobby"|"room"|"config");

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Narrowed shape of a StateDocument. */
        type $Shape = {
          lobby?: game.LobbySnapshot.$Shape|null;
          room?: game.RoomSnapshot.$Shape|null;
          config?: game.AppConfig.$Shape|null;
          $unknowns?: Uint8Array[];
        } & (
          ({ doc?: undefined; lobby?: null; room?: null; config?: null }|{ doc?: "lobby"; lobby: game.LobbySnapshot.$Shape; room?: null; config?: null }|{ doc?: "room"; lobby?: null; room: game.RoomSnapshot.$Shape; config?: null }|{ doc?: "config"; lobby?: null; room?: null; config: game.AppConfig.$Shape })
        );
    }

    /**
     * Properties of a PlayerBatch.
     * @deprecated Use game.PlayerBatch.$Properties instead.
     */
    interface IPlayerBatch extends game.PlayerBatch.$Properties {
    }

    /** Represents a PlayerBatch. */
    class PlayerBatch {

        /**
         * Constructs a new PlayerBatch.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.PlayerBatch.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** PlayerBatch players. */
        players: game.PublicPlayer.$Properties[];

        /**
         * Creates a new PlayerBatch instance using the specified properties.
         * @param [properties] Properties to set
         * @returns PlayerBatch instance
         */
        static create(properties: game.PlayerBatch.$Shape): game.PlayerBatch & game.PlayerBatch.$Shape;
        static create(properties?: game.PlayerBatch.$Properties): game.PlayerBatch;

        /**
         * Encodes the specified PlayerBatch message. Does not implicitly {@link game.PlayerBatch.verify|verify} messages.
         * @param message PlayerBatch message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.PlayerBatch.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified PlayerBatch message, length delimited. Does not implicitly {@link game.PlayerBatch.verify|verify} messages.
         * @param message PlayerBatch message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.PlayerBatch.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a PlayerBatch message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.PlayerBatch & game.PlayerBatch.$Shape} PlayerBatch
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.PlayerBatch & game.PlayerBatch.$Shape;

        /**
         * Decodes a PlayerBatch message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.PlayerBatch & game.PlayerBatch.$Shape} PlayerBatch
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.PlayerBatch & game.PlayerBatch.$Shape;

        /**
         * Verifies a PlayerBatch message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a PlayerBatch message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns PlayerBatch
         */
        static fromObject(object: { [k: string]: any }): game.PlayerBatch;

        /**
         * Creates a plain object from a PlayerBatch message. Also converts values to other types if specified.
         * @param message PlayerBatch
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.PlayerBatch, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this PlayerBatch to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for PlayerBatch
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace PlayerBatch {

        /** Properties of a PlayerBatch. */
        interface $Properties {

            /** PlayerBatch players */
            players?: (game.PublicPlayer.$Properties[]|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a PlayerBatch. */
        type $Shape = game.PlayerBatch.$Properties;
    }

    /**
     * Properties of a MeState.
     * @deprecated Use game.MeState.$Properties instead.
     */
    interface IMeState extends game.MeState.$Properties {
    }

    /** Represents a MeState. */
    class MeState {

        /**
         * Constructs a new MeState.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.MeState.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** MeState player. */
        player?: (game.PublicPlayer.$Properties|null);

        /** MeState token. */
        token: string;

        /** MeState roomId. */
        roomId: string;

        /** MeState room. */
        room?: (game.RoomSnapshot.$Properties|null);

        /**
         * Creates a new MeState instance using the specified properties.
         * @param [properties] Properties to set
         * @returns MeState instance
         */
        static create(properties: game.MeState.$Shape): game.MeState & game.MeState.$Shape;
        static create(properties?: game.MeState.$Properties): game.MeState;

        /**
         * Encodes the specified MeState message. Does not implicitly {@link game.MeState.verify|verify} messages.
         * @param message MeState message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.MeState.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified MeState message, length delimited. Does not implicitly {@link game.MeState.verify|verify} messages.
         * @param message MeState message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.MeState.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a MeState message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.MeState & game.MeState.$Shape} MeState
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.MeState & game.MeState.$Shape;

        /**
         * Decodes a MeState message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.MeState & game.MeState.$Shape} MeState
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.MeState & game.MeState.$Shape;

        /**
         * Verifies a MeState message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a MeState message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns MeState
         */
        static fromObject(object: { [k: string]: any }): game.MeState;

        /**
         * Creates a plain object from a MeState message. Also converts values to other types if specified.
         * @param message MeState
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.MeState, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this MeState to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for MeState
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace MeState {

        /** Properties of a MeState. */
        interface $Properties {

            /** MeState player */
            player?: (game.PublicPlayer.$Properties|null);

            /** MeState token */
            token?: (string|null);

            /** MeState roomId */
            roomId?: (string|null);

            /** MeState room */
            room?: (game.RoomSnapshot.$Properties|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a MeState. */
        type $Shape = {
          player?: game.PublicPlayer.$Shape|null;
          token?: string|null;
          roomId?: string|null;
          room?: game.RoomSnapshot.$Shape|null;
          $unknowns?: Uint8Array[];
        };
    }

    /**
     * Properties of an AnnouncementPayload.
     * @deprecated Use game.AnnouncementPayload.$Properties instead.
     */
    interface IAnnouncementPayload extends game.AnnouncementPayload.$Properties {
    }

    /** Represents an AnnouncementPayload. */
    class AnnouncementPayload {

        /**
         * Constructs a new AnnouncementPayload.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.AnnouncementPayload.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** AnnouncementPayload id. */
        id: string;

        /** AnnouncementPayload message. */
        message: string;

        /** AnnouncementPayload durationMs. */
        durationMs: number;

        /** AnnouncementPayload createdAt. */
        createdAt: (number|Long);

        /**
         * Creates a new AnnouncementPayload instance using the specified properties.
         * @param [properties] Properties to set
         * @returns AnnouncementPayload instance
         */
        static create(properties: game.AnnouncementPayload.$Shape): game.AnnouncementPayload & game.AnnouncementPayload.$Shape;
        static create(properties?: game.AnnouncementPayload.$Properties): game.AnnouncementPayload;

        /**
         * Encodes the specified AnnouncementPayload message. Does not implicitly {@link game.AnnouncementPayload.verify|verify} messages.
         * @param message AnnouncementPayload message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.AnnouncementPayload.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified AnnouncementPayload message, length delimited. Does not implicitly {@link game.AnnouncementPayload.verify|verify} messages.
         * @param message AnnouncementPayload message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.AnnouncementPayload.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes an AnnouncementPayload message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.AnnouncementPayload & game.AnnouncementPayload.$Shape} AnnouncementPayload
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.AnnouncementPayload & game.AnnouncementPayload.$Shape;

        /**
         * Decodes an AnnouncementPayload message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.AnnouncementPayload & game.AnnouncementPayload.$Shape} AnnouncementPayload
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.AnnouncementPayload & game.AnnouncementPayload.$Shape;

        /**
         * Verifies an AnnouncementPayload message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates an AnnouncementPayload message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns AnnouncementPayload
         */
        static fromObject(object: { [k: string]: any }): game.AnnouncementPayload;

        /**
         * Creates a plain object from an AnnouncementPayload message. Also converts values to other types if specified.
         * @param message AnnouncementPayload
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.AnnouncementPayload, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this AnnouncementPayload to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for AnnouncementPayload
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace AnnouncementPayload {

        /** Properties of an AnnouncementPayload. */
        interface $Properties {

            /** AnnouncementPayload id */
            id?: (string|null);

            /** AnnouncementPayload message */
            message?: (string|null);

            /** AnnouncementPayload durationMs */
            durationMs?: (number|null);

            /** AnnouncementPayload createdAt */
            createdAt?: (number|Long|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of an AnnouncementPayload. */
        type $Shape = game.AnnouncementPayload.$Properties;
    }

    /**
     * Properties of a RoomClosed.
     * @deprecated Use game.RoomClosed.$Properties instead.
     */
    interface IRoomClosed extends game.RoomClosed.$Properties {
    }

    /** Represents a RoomClosed. */
    class RoomClosed {

        /**
         * Constructs a new RoomClosed.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.RoomClosed.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** RoomClosed message. */
        message: string;

        /**
         * Creates a new RoomClosed instance using the specified properties.
         * @param [properties] Properties to set
         * @returns RoomClosed instance
         */
        static create(properties: game.RoomClosed.$Shape): game.RoomClosed & game.RoomClosed.$Shape;
        static create(properties?: game.RoomClosed.$Properties): game.RoomClosed;

        /**
         * Encodes the specified RoomClosed message. Does not implicitly {@link game.RoomClosed.verify|verify} messages.
         * @param message RoomClosed message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.RoomClosed.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified RoomClosed message, length delimited. Does not implicitly {@link game.RoomClosed.verify|verify} messages.
         * @param message RoomClosed message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.RoomClosed.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a RoomClosed message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.RoomClosed & game.RoomClosed.$Shape} RoomClosed
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.RoomClosed & game.RoomClosed.$Shape;

        /**
         * Decodes a RoomClosed message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.RoomClosed & game.RoomClosed.$Shape} RoomClosed
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.RoomClosed & game.RoomClosed.$Shape;

        /**
         * Verifies a RoomClosed message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a RoomClosed message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns RoomClosed
         */
        static fromObject(object: { [k: string]: any }): game.RoomClosed;

        /**
         * Creates a plain object from a RoomClosed message. Also converts values to other types if specified.
         * @param message RoomClosed
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.RoomClosed, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this RoomClosed to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for RoomClosed
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace RoomClosed {

        /** Properties of a RoomClosed. */
        interface $Properties {

            /** RoomClosed message */
            message?: (string|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a RoomClosed. */
        type $Shape = game.RoomClosed.$Properties;
    }

    /**
     * Properties of a HistoryPage.
     * @deprecated Use game.HistoryPage.$Properties instead.
     */
    interface IHistoryPage extends game.HistoryPage.$Properties {
    }

    /** Represents a HistoryPage. */
    class HistoryPage {

        /**
         * Constructs a new HistoryPage.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.HistoryPage.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** HistoryPage roomId. */
        roomId: string;

        /** HistoryPage item. */
        item?: (game.RoundHistoryItem.$Properties|null);

        /** HistoryPage total. */
        total: number;

        /**
         * Creates a new HistoryPage instance using the specified properties.
         * @param [properties] Properties to set
         * @returns HistoryPage instance
         */
        static create(properties: game.HistoryPage.$Shape): game.HistoryPage & game.HistoryPage.$Shape;
        static create(properties?: game.HistoryPage.$Properties): game.HistoryPage;

        /**
         * Encodes the specified HistoryPage message. Does not implicitly {@link game.HistoryPage.verify|verify} messages.
         * @param message HistoryPage message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.HistoryPage.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified HistoryPage message, length delimited. Does not implicitly {@link game.HistoryPage.verify|verify} messages.
         * @param message HistoryPage message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.HistoryPage.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a HistoryPage message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.HistoryPage & game.HistoryPage.$Shape} HistoryPage
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.HistoryPage & game.HistoryPage.$Shape;

        /**
         * Decodes a HistoryPage message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.HistoryPage & game.HistoryPage.$Shape} HistoryPage
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.HistoryPage & game.HistoryPage.$Shape;

        /**
         * Verifies a HistoryPage message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a HistoryPage message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns HistoryPage
         */
        static fromObject(object: { [k: string]: any }): game.HistoryPage;

        /**
         * Creates a plain object from a HistoryPage message. Also converts values to other types if specified.
         * @param message HistoryPage
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.HistoryPage, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this HistoryPage to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for HistoryPage
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace HistoryPage {

        /** Properties of a HistoryPage. */
        interface $Properties {

            /** HistoryPage roomId */
            roomId?: (string|null);

            /** HistoryPage item */
            item?: (game.RoundHistoryItem.$Properties|null);

            /** HistoryPage total */
            total?: (number|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a HistoryPage. */
        type $Shape = game.HistoryPage.$Properties;
    }

    /**
     * Properties of an OkResult.
     * @deprecated Use game.OkResult.$Properties instead.
     */
    interface IOkResult extends game.OkResult.$Properties {
    }

    /** Represents an OkResult. */
    class OkResult {

        /**
         * Constructs a new OkResult.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.OkResult.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** OkResult ok. */
        ok: boolean;

        /**
         * Creates a new OkResult instance using the specified properties.
         * @param [properties] Properties to set
         * @returns OkResult instance
         */
        static create(properties: game.OkResult.$Shape): game.OkResult & game.OkResult.$Shape;
        static create(properties?: game.OkResult.$Properties): game.OkResult;

        /**
         * Encodes the specified OkResult message. Does not implicitly {@link game.OkResult.verify|verify} messages.
         * @param message OkResult message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.OkResult.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified OkResult message, length delimited. Does not implicitly {@link game.OkResult.verify|verify} messages.
         * @param message OkResult message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.OkResult.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes an OkResult message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.OkResult & game.OkResult.$Shape} OkResult
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.OkResult & game.OkResult.$Shape;

        /**
         * Decodes an OkResult message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.OkResult & game.OkResult.$Shape} OkResult
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.OkResult & game.OkResult.$Shape;

        /**
         * Verifies an OkResult message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates an OkResult message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns OkResult
         */
        static fromObject(object: { [k: string]: any }): game.OkResult;

        /**
         * Creates a plain object from an OkResult message. Also converts values to other types if specified.
         * @param message OkResult
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.OkResult, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this OkResult to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for OkResult
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace OkResult {

        /** Properties of an OkResult. */
        interface $Properties {

            /** OkResult ok */
            ok?: (boolean|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of an OkResult. */
        type $Shape = game.OkResult.$Properties;
    }

    /**
     * Properties of a PlayerResult.
     * @deprecated Use game.PlayerResult.$Properties instead.
     */
    interface IPlayerResult extends game.PlayerResult.$Properties {
    }

    /** Represents a PlayerResult. */
    class PlayerResult {

        /**
         * Constructs a new PlayerResult.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.PlayerResult.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** PlayerResult player. */
        player?: (game.PublicPlayer.$Properties|null);

        /**
         * Creates a new PlayerResult instance using the specified properties.
         * @param [properties] Properties to set
         * @returns PlayerResult instance
         */
        static create(properties: game.PlayerResult.$Shape): game.PlayerResult & game.PlayerResult.$Shape;
        static create(properties?: game.PlayerResult.$Properties): game.PlayerResult;

        /**
         * Encodes the specified PlayerResult message. Does not implicitly {@link game.PlayerResult.verify|verify} messages.
         * @param message PlayerResult message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.PlayerResult.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified PlayerResult message, length delimited. Does not implicitly {@link game.PlayerResult.verify|verify} messages.
         * @param message PlayerResult message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.PlayerResult.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a PlayerResult message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.PlayerResult & game.PlayerResult.$Shape} PlayerResult
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.PlayerResult & game.PlayerResult.$Shape;

        /**
         * Decodes a PlayerResult message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.PlayerResult & game.PlayerResult.$Shape} PlayerResult
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.PlayerResult & game.PlayerResult.$Shape;

        /**
         * Verifies a PlayerResult message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a PlayerResult message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns PlayerResult
         */
        static fromObject(object: { [k: string]: any }): game.PlayerResult;

        /**
         * Creates a plain object from a PlayerResult message. Also converts values to other types if specified.
         * @param message PlayerResult
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.PlayerResult, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this PlayerResult to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for PlayerResult
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace PlayerResult {

        /** Properties of a PlayerResult. */
        interface $Properties {

            /** PlayerResult player */
            player?: (game.PublicPlayer.$Properties|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a PlayerResult. */
        type $Shape = game.PlayerResult.$Properties;
    }

    /**
     * Properties of a SuggestionsResult.
     * @deprecated Use game.SuggestionsResult.$Properties instead.
     */
    interface ISuggestionsResult extends game.SuggestionsResult.$Properties {
    }

    /** Represents a SuggestionsResult. */
    class SuggestionsResult {

        /**
         * Constructs a new SuggestionsResult.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.SuggestionsResult.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** SuggestionsResult suggestions. */
        suggestions: game.Suggestion.$Properties[];

        /**
         * Creates a new SuggestionsResult instance using the specified properties.
         * @param [properties] Properties to set
         * @returns SuggestionsResult instance
         */
        static create(properties: game.SuggestionsResult.$Shape): game.SuggestionsResult & game.SuggestionsResult.$Shape;
        static create(properties?: game.SuggestionsResult.$Properties): game.SuggestionsResult;

        /**
         * Encodes the specified SuggestionsResult message. Does not implicitly {@link game.SuggestionsResult.verify|verify} messages.
         * @param message SuggestionsResult message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.SuggestionsResult.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified SuggestionsResult message, length delimited. Does not implicitly {@link game.SuggestionsResult.verify|verify} messages.
         * @param message SuggestionsResult message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.SuggestionsResult.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a SuggestionsResult message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.SuggestionsResult & game.SuggestionsResult.$Shape} SuggestionsResult
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.SuggestionsResult & game.SuggestionsResult.$Shape;

        /**
         * Decodes a SuggestionsResult message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.SuggestionsResult & game.SuggestionsResult.$Shape} SuggestionsResult
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.SuggestionsResult & game.SuggestionsResult.$Shape;

        /**
         * Verifies a SuggestionsResult message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a SuggestionsResult message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns SuggestionsResult
         */
        static fromObject(object: { [k: string]: any }): game.SuggestionsResult;

        /**
         * Creates a plain object from a SuggestionsResult message. Also converts values to other types if specified.
         * @param message SuggestionsResult
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.SuggestionsResult, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this SuggestionsResult to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for SuggestionsResult
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace SuggestionsResult {

        /** Properties of a SuggestionsResult. */
        interface $Properties {

            /** SuggestionsResult suggestions */
            suggestions?: (game.Suggestion.$Properties[]|null);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Shape of a SuggestionsResult. */
        type $Shape = game.SuggestionsResult.$Properties;
    }

    /**
     * Properties of a RawBody.
     * @deprecated Use game.RawBody.$Properties instead.
     */
    interface IRawBody extends game.RawBody.$Properties {
    }

    /** Represents a RawBody. */
    class RawBody {

        /**
         * Constructs a new RawBody.
         * @param [properties] Properties to set
         */
        constructor(properties?: game.RawBody.$Properties);

        /** Unknown fields preserved while decoding when enabled */
        $unknowns?: Uint8Array[];

        /** RawBody dynamic. */
        dynamic?: (google.protobuf.Struct.$Properties|null);

        /** RawBody playerBatch. */
        playerBatch?: (game.PlayerBatch.$Properties|null);

        /** RawBody chat. */
        chat?: (game.ChatMessage.$Properties|null);

        /** RawBody suggestion. */
        suggestion?: (game.Suggestion.$Properties|null);

        /** RawBody player. */
        player?: (game.PublicPlayer.$Properties|null);

        /** RawBody me. */
        me?: (game.MeState.$Properties|null);

        /** RawBody announcement. */
        announcement?: (game.AnnouncementPayload.$Properties|null);

        /** RawBody roomClosed. */
        roomClosed?: (game.RoomClosed.$Properties|null);

        /** RawBody historyPage. */
        historyPage?: (game.HistoryPage.$Properties|null);

        /** RawBody ok. */
        ok?: (game.OkResult.$Properties|null);

        /** RawBody playerResult. */
        playerResult?: (game.PlayerResult.$Properties|null);

        /** RawBody suggestions. */
        suggestions?: (game.SuggestionsResult.$Properties|null);

        /** RawBody room. */
        room?: (game.RoomSnapshot.$Properties|null);

        /** RawBody config. */
        config?: (game.AppConfig.$Properties|null);

        /** RawBody lobby. */
        lobby?: (game.LobbySnapshot.$Properties|null);

        /** RawBody body. */
        body?: ("dynamic"|"playerBatch"|"chat"|"suggestion"|"player"|"me"|"announcement"|"roomClosed"|"historyPage"|"ok"|"playerResult"|"suggestions"|"room"|"config"|"lobby");

        /**
         * Creates a new RawBody instance using the specified properties.
         * @param [properties] Properties to set
         * @returns RawBody instance
         */
        static create(properties: game.RawBody.$Shape): game.RawBody & game.RawBody.$Shape;
        static create(properties?: game.RawBody.$Properties): game.RawBody;

        /**
         * Encodes the specified RawBody message. Does not implicitly {@link game.RawBody.verify|verify} messages.
         * @param message RawBody message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encode(message: game.RawBody.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified RawBody message, length delimited. Does not implicitly {@link game.RawBody.verify|verify} messages.
         * @param message RawBody message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        static encodeDelimited(message: game.RawBody.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a RawBody message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns {game.RawBody & game.RawBody.$Shape} RawBody
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): game.RawBody & game.RawBody.$Shape;

        /**
         * Decodes a RawBody message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns {game.RawBody & game.RawBody.$Shape} RawBody
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): game.RawBody & game.RawBody.$Shape;

        /**
         * Verifies a RawBody message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a RawBody message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns RawBody
         */
        static fromObject(object: { [k: string]: any }): game.RawBody;

        /**
         * Creates a plain object from a RawBody message. Also converts values to other types if specified.
         * @param message RawBody
         * @param [options] Conversion options
         * @returns Plain object
         */
        static toObject(message: game.RawBody, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this RawBody to JSON.
         * @returns JSON object
         */
        toJSON(): { [k: string]: any };

        /**
         * Gets the type url for RawBody
         * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
         * @returns The type url
         */
        static getTypeUrl(prefix?: string): string;
    }

    namespace RawBody {

        /** Properties of a RawBody. */
        interface $Properties {

            /** RawBody dynamic */
            dynamic?: (google.protobuf.Struct.$Properties|null);

            /** RawBody playerBatch */
            playerBatch?: (game.PlayerBatch.$Properties|null);

            /** RawBody chat */
            chat?: (game.ChatMessage.$Properties|null);

            /** RawBody suggestion */
            suggestion?: (game.Suggestion.$Properties|null);

            /** RawBody player */
            player?: (game.PublicPlayer.$Properties|null);

            /** RawBody me */
            me?: (game.MeState.$Properties|null);

            /** RawBody announcement */
            announcement?: (game.AnnouncementPayload.$Properties|null);

            /** RawBody roomClosed */
            roomClosed?: (game.RoomClosed.$Properties|null);

            /** RawBody historyPage */
            historyPage?: (game.HistoryPage.$Properties|null);

            /** RawBody ok */
            ok?: (game.OkResult.$Properties|null);

            /** RawBody playerResult */
            playerResult?: (game.PlayerResult.$Properties|null);

            /** RawBody suggestions */
            suggestions?: (game.SuggestionsResult.$Properties|null);

            /** RawBody room */
            room?: (game.RoomSnapshot.$Properties|null);

            /** RawBody config */
            config?: (game.AppConfig.$Properties|null);

            /** RawBody lobby */
            lobby?: (game.LobbySnapshot.$Properties|null);

            /** RawBody body */
            body?: ("dynamic"|"playerBatch"|"chat"|"suggestion"|"player"|"me"|"announcement"|"roomClosed"|"historyPage"|"ok"|"playerResult"|"suggestions"|"room"|"config"|"lobby");

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];
        }

        /** Narrowed shape of a RawBody. */
        type $Shape = {
          dynamic?: google.protobuf.Struct.$Shape|null;
          playerBatch?: game.PlayerBatch.$Shape|null;
          chat?: game.ChatMessage.$Shape|null;
          suggestion?: game.Suggestion.$Shape|null;
          player?: game.PublicPlayer.$Shape|null;
          me?: game.MeState.$Shape|null;
          announcement?: game.AnnouncementPayload.$Shape|null;
          roomClosed?: game.RoomClosed.$Shape|null;
          historyPage?: game.HistoryPage.$Shape|null;
          ok?: game.OkResult.$Shape|null;
          playerResult?: game.PlayerResult.$Shape|null;
          suggestions?: game.SuggestionsResult.$Shape|null;
          room?: game.RoomSnapshot.$Shape|null;
          config?: game.AppConfig.$Shape|null;
          lobby?: game.LobbySnapshot.$Shape|null;
          $unknowns?: Uint8Array[];
        } & (
          ({ body?: undefined; dynamic?: null; playerBatch?: null; chat?: null; suggestion?: null; player?: null; me?: null; announcement?: null; roomClosed?: null; historyPage?: null; ok?: null; playerResult?: null; suggestions?: null; room?: null; config?: null; lobby?: null }|{ body?: "dynamic"; dynamic: google.protobuf.Struct.$Shape; playerBatch?: null; chat?: null; suggestion?: null; player?: null; me?: null; announcement?: null; roomClosed?: null; historyPage?: null; ok?: null; playerResult?: null; suggestions?: null; room?: null; config?: null; lobby?: null }|{ body?: "playerBatch"; dynamic?: null; playerBatch: game.PlayerBatch.$Shape; chat?: null; suggestion?: null; player?: null; me?: null; announcement?: null; roomClosed?: null; historyPage?: null; ok?: null; playerResult?: null; suggestions?: null; room?: null; config?: null; lobby?: null }|{ body?: "chat"; dynamic?: null; playerBatch?: null; chat: game.ChatMessage.$Shape; suggestion?: null; player?: null; me?: null; announcement?: null; roomClosed?: null; historyPage?: null; ok?: null; playerResult?: null; suggestions?: null; room?: null; config?: null; lobby?: null }|{ body?: "suggestion"; dynamic?: null; playerBatch?: null; chat?: null; suggestion: game.Suggestion.$Shape; player?: null; me?: null; announcement?: null; roomClosed?: null; historyPage?: null; ok?: null; playerResult?: null; suggestions?: null; room?: null; config?: null; lobby?: null }|{ body?: "player"; dynamic?: null; playerBatch?: null; chat?: null; suggestion?: null; player: game.PublicPlayer.$Shape; me?: null; announcement?: null; roomClosed?: null; historyPage?: null; ok?: null; playerResult?: null; suggestions?: null; room?: null; config?: null; lobby?: null }|{ body?: "me"; dynamic?: null; playerBatch?: null; chat?: null; suggestion?: null; player?: null; me: game.MeState.$Shape; announcement?: null; roomClosed?: null; historyPage?: null; ok?: null; playerResult?: null; suggestions?: null; room?: null; config?: null; lobby?: null }|{ body?: "announcement"; dynamic?: null; playerBatch?: null; chat?: null; suggestion?: null; player?: null; me?: null; announcement: game.AnnouncementPayload.$Shape; roomClosed?: null; historyPage?: null; ok?: null; playerResult?: null; suggestions?: null; room?: null; config?: null; lobby?: null }|{ body?: "roomClosed"; dynamic?: null; playerBatch?: null; chat?: null; suggestion?: null; player?: null; me?: null; announcement?: null; roomClosed: game.RoomClosed.$Shape; historyPage?: null; ok?: null; playerResult?: null; suggestions?: null; room?: null; config?: null; lobby?: null }|{ body?: "historyPage"; dynamic?: null; playerBatch?: null; chat?: null; suggestion?: null; player?: null; me?: null; announcement?: null; roomClosed?: null; historyPage: game.HistoryPage.$Shape; ok?: null; playerResult?: null; suggestions?: null; room?: null; config?: null; lobby?: null }|{ body?: "ok"; dynamic?: null; playerBatch?: null; chat?: null; suggestion?: null; player?: null; me?: null; announcement?: null; roomClosed?: null; historyPage?: null; ok: game.OkResult.$Shape; playerResult?: null; suggestions?: null; room?: null; config?: null; lobby?: null }|{ body?: "playerResult"; dynamic?: null; playerBatch?: null; chat?: null; suggestion?: null; player?: null; me?: null; announcement?: null; roomClosed?: null; historyPage?: null; ok?: null; playerResult: game.PlayerResult.$Shape; suggestions?: null; room?: null; config?: null; lobby?: null }|{ body?: "suggestions"; dynamic?: null; playerBatch?: null; chat?: null; suggestion?: null; player?: null; me?: null; announcement?: null; roomClosed?: null; historyPage?: null; ok?: null; playerResult?: null; suggestions: game.SuggestionsResult.$Shape; room?: null; config?: null; lobby?: null }|{ body?: "room"; dynamic?: null; playerBatch?: null; chat?: null; suggestion?: null; player?: null; me?: null; announcement?: null; roomClosed?: null; historyPage?: null; ok?: null; playerResult?: null; suggestions?: null; room: game.RoomSnapshot.$Shape; config?: null; lobby?: null }|{ body?: "config"; dynamic?: null; playerBatch?: null; chat?: null; suggestion?: null; player?: null; me?: null; announcement?: null; roomClosed?: null; historyPage?: null; ok?: null; playerResult?: null; suggestions?: null; room?: null; config: game.AppConfig.$Shape; lobby?: null }|{ body?: "lobby"; dynamic?: null; playerBatch?: null; chat?: null; suggestion?: null; player?: null; me?: null; announcement?: null; roomClosed?: null; historyPage?: null; ok?: null; playerResult?: null; suggestions?: null; room?: null; config?: null; lobby: game.LobbySnapshot.$Shape })
        );
    }
}

/** Namespace google. */
export namespace google {

    /** Namespace protobuf. */
    namespace protobuf {

        /**
         * Properties of a Struct.
         * @deprecated Use google.protobuf.Struct.$Properties instead.
         */
        interface IStruct extends google.protobuf.Struct.$Properties {
        }

        /** Represents a Struct. */
        class Struct {

            /**
             * Constructs a new Struct.
             * @param [properties] Properties to set
             */
            constructor(properties?: google.protobuf.Struct.$Properties);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];

            /** Struct fields. */
            fields: { [k: string]: google.protobuf.Value.$Properties };

            /**
             * Creates a new Struct instance using the specified properties.
             * @param [properties] Properties to set
             * @returns Struct instance
             */
            static create(properties: google.protobuf.Struct.$Shape): google.protobuf.Struct & google.protobuf.Struct.$Shape;
            static create(properties?: google.protobuf.Struct.$Properties): google.protobuf.Struct;

            /**
             * Encodes the specified Struct message. Does not implicitly {@link google.protobuf.Struct.verify|verify} messages.
             * @param message Struct message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            static encode(message: google.protobuf.Struct.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified Struct message, length delimited. Does not implicitly {@link google.protobuf.Struct.verify|verify} messages.
             * @param message Struct message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            static encodeDelimited(message: google.protobuf.Struct.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a Struct message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns {google.protobuf.Struct & google.protobuf.Struct.$Shape} Struct
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.Struct & google.protobuf.Struct.$Shape;

            /**
             * Decodes a Struct message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns {google.protobuf.Struct & google.protobuf.Struct.$Shape} Struct
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.Struct & google.protobuf.Struct.$Shape;

            /**
             * Verifies a Struct message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates a Struct message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns Struct
             */
            static fromObject(object: { [k: string]: any }): google.protobuf.Struct;

            /**
             * Creates a plain object from a Struct message. Also converts values to other types if specified.
             * @param message Struct
             * @param [options] Conversion options
             * @returns Plain object
             */
            static toObject(message: google.protobuf.Struct, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this Struct to JSON.
             * @returns JSON object
             */
            toJSON(): { [k: string]: any };

            /**
             * Gets the type url for Struct
             * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
             * @returns The type url
             */
            static getTypeUrl(prefix?: string): string;
        }

        namespace Struct {

            /** Properties of a Struct. */
            interface $Properties {

                /** Struct fields */
                fields?: ({ [k: string]: google.protobuf.Value.$Properties }|null);

                /** Unknown fields preserved while decoding when enabled */
                $unknowns?: Uint8Array[];
            }

            /** Shape of a Struct. */
            type $Shape = {
              fields?: { [k: string]: google.protobuf.Value.$Shape }|null;
              $unknowns?: Uint8Array[];
            };
        }

        /**
         * Properties of a Value.
         * @deprecated Use google.protobuf.Value.$Properties instead.
         */
        interface IValue extends google.protobuf.Value.$Properties {
        }

        /** Represents a Value. */
        class Value {

            /**
             * Constructs a new Value.
             * @param [properties] Properties to set
             */
            constructor(properties?: google.protobuf.Value.$Properties);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];

            /** Value nullValue. */
            nullValue?: (google.protobuf.NullValue|null);

            /** Value numberValue. */
            numberValue?: (number|null);

            /** Value stringValue. */
            stringValue?: (string|null);

            /** Value boolValue. */
            boolValue?: (boolean|null);

            /** Value structValue. */
            structValue?: (google.protobuf.Struct.$Properties|null);

            /** Value listValue. */
            listValue?: (google.protobuf.ListValue.$Properties|null);

            /** Value kind. */
            kind?: ("nullValue"|"numberValue"|"stringValue"|"boolValue"|"structValue"|"listValue");

            /**
             * Creates a new Value instance using the specified properties.
             * @param [properties] Properties to set
             * @returns Value instance
             */
            static create(properties: google.protobuf.Value.$Shape): google.protobuf.Value & google.protobuf.Value.$Shape;
            static create(properties?: google.protobuf.Value.$Properties): google.protobuf.Value;

            /**
             * Encodes the specified Value message. Does not implicitly {@link google.protobuf.Value.verify|verify} messages.
             * @param message Value message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            static encode(message: google.protobuf.Value.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified Value message, length delimited. Does not implicitly {@link google.protobuf.Value.verify|verify} messages.
             * @param message Value message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            static encodeDelimited(message: google.protobuf.Value.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a Value message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns {google.protobuf.Value & google.protobuf.Value.$Shape} Value
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.Value & google.protobuf.Value.$Shape;

            /**
             * Decodes a Value message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns {google.protobuf.Value & google.protobuf.Value.$Shape} Value
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.Value & google.protobuf.Value.$Shape;

            /**
             * Verifies a Value message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates a Value message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns Value
             */
            static fromObject(object: { [k: string]: any }): google.protobuf.Value;

            /**
             * Creates a plain object from a Value message. Also converts values to other types if specified.
             * @param message Value
             * @param [options] Conversion options
             * @returns Plain object
             */
            static toObject(message: google.protobuf.Value, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this Value to JSON.
             * @returns JSON object
             */
            toJSON(): { [k: string]: any };

            /**
             * Gets the type url for Value
             * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
             * @returns The type url
             */
            static getTypeUrl(prefix?: string): string;
        }

        namespace Value {

            /** Properties of a Value. */
            interface $Properties {

                /** Value nullValue */
                nullValue?: (google.protobuf.NullValue|null);

                /** Value numberValue */
                numberValue?: (number|null);

                /** Value stringValue */
                stringValue?: (string|null);

                /** Value boolValue */
                boolValue?: (boolean|null);

                /** Value structValue */
                structValue?: (google.protobuf.Struct.$Properties|null);

                /** Value listValue */
                listValue?: (google.protobuf.ListValue.$Properties|null);

                /** Value kind */
                kind?: ("nullValue"|"numberValue"|"stringValue"|"boolValue"|"structValue"|"listValue");

                /** Unknown fields preserved while decoding when enabled */
                $unknowns?: Uint8Array[];
            }

            /** Narrowed shape of a Value. */
            type $Shape = {
              nullValue?: google.protobuf.NullValue|null;
              numberValue?: number|null;
              stringValue?: string|null;
              boolValue?: boolean|null;
              structValue?: google.protobuf.Struct.$Shape|null;
              listValue?: google.protobuf.ListValue.$Shape|null;
              $unknowns?: Uint8Array[];
            } & (
              ({ kind?: undefined; nullValue?: null; numberValue?: null; stringValue?: null; boolValue?: null; structValue?: null; listValue?: null }|{ kind?: "nullValue"; nullValue: google.protobuf.NullValue; numberValue?: null; stringValue?: null; boolValue?: null; structValue?: null; listValue?: null }|{ kind?: "numberValue"; nullValue?: null; numberValue: number; stringValue?: null; boolValue?: null; structValue?: null; listValue?: null }|{ kind?: "stringValue"; nullValue?: null; numberValue?: null; stringValue: string; boolValue?: null; structValue?: null; listValue?: null }|{ kind?: "boolValue"; nullValue?: null; numberValue?: null; stringValue?: null; boolValue: boolean; structValue?: null; listValue?: null }|{ kind?: "structValue"; nullValue?: null; numberValue?: null; stringValue?: null; boolValue?: null; structValue: google.protobuf.Struct.$Shape; listValue?: null }|{ kind?: "listValue"; nullValue?: null; numberValue?: null; stringValue?: null; boolValue?: null; structValue?: null; listValue: google.protobuf.ListValue.$Shape })
            );
        }

        /** NullValue enum. */
        enum NullValue {

            /** NULL_VALUE value */
            NULL_VALUE = 0
        }

        /**
         * Properties of a ListValue.
         * @deprecated Use google.protobuf.ListValue.$Properties instead.
         */
        interface IListValue extends google.protobuf.ListValue.$Properties {
        }

        /** Represents a ListValue. */
        class ListValue {

            /**
             * Constructs a new ListValue.
             * @param [properties] Properties to set
             */
            constructor(properties?: google.protobuf.ListValue.$Properties);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];

            /** ListValue values. */
            values: google.protobuf.Value.$Properties[];

            /**
             * Creates a new ListValue instance using the specified properties.
             * @param [properties] Properties to set
             * @returns ListValue instance
             */
            static create(properties: google.protobuf.ListValue.$Shape): google.protobuf.ListValue & google.protobuf.ListValue.$Shape;
            static create(properties?: google.protobuf.ListValue.$Properties): google.protobuf.ListValue;

            /**
             * Encodes the specified ListValue message. Does not implicitly {@link google.protobuf.ListValue.verify|verify} messages.
             * @param message ListValue message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            static encode(message: google.protobuf.ListValue.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified ListValue message, length delimited. Does not implicitly {@link google.protobuf.ListValue.verify|verify} messages.
             * @param message ListValue message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            static encodeDelimited(message: google.protobuf.ListValue.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a ListValue message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns {google.protobuf.ListValue & google.protobuf.ListValue.$Shape} ListValue
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.ListValue & google.protobuf.ListValue.$Shape;

            /**
             * Decodes a ListValue message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns {google.protobuf.ListValue & google.protobuf.ListValue.$Shape} ListValue
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.ListValue & google.protobuf.ListValue.$Shape;

            /**
             * Verifies a ListValue message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates a ListValue message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns ListValue
             */
            static fromObject(object: { [k: string]: any }): google.protobuf.ListValue;

            /**
             * Creates a plain object from a ListValue message. Also converts values to other types if specified.
             * @param message ListValue
             * @param [options] Conversion options
             * @returns Plain object
             */
            static toObject(message: google.protobuf.ListValue, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this ListValue to JSON.
             * @returns JSON object
             */
            toJSON(): { [k: string]: any };

            /**
             * Gets the type url for ListValue
             * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
             * @returns The type url
             */
            static getTypeUrl(prefix?: string): string;
        }

        namespace ListValue {

            /** Properties of a ListValue. */
            interface $Properties {

                /** ListValue values */
                values?: (google.protobuf.Value.$Properties[]|null);

                /** Unknown fields preserved while decoding when enabled */
                $unknowns?: Uint8Array[];
            }

            /** Shape of a ListValue. */
            type $Shape = {
              values?: google.protobuf.Value.$Shape[]|null;
              $unknowns?: Uint8Array[];
            };
        }
    }
}
