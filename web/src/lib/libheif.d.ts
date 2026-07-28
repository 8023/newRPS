declare module "libheif-js/libheif-wasm/libheif.wasm?url" {
  const url: string;
  export default url;
}

declare module "libheif-js/libheif-wasm/libheif.js" {
  export interface HeifImage {
    get_width(): number;
    get_height(): number;
    display(imageData: ImageData, callback: (result: ImageData | null) => void): void;
    free(): void;
  }
  export interface HeifDecoder {
    decode(buffer: Uint8Array): HeifImage[];
  }
  export interface HeifModule {
    HeifDecoder: new () => HeifDecoder;
  }
  export default function createHeifModule(options?: {
    wasmBinary?: Uint8Array;
    locateFile?: (path: string, prefix: string) => string;
  }): Promise<HeifModule>;
}
