declare module 'emoji-mart-vue-fast' {
  import { DefineComponent } from 'vue';

  export interface EmojiData {
    id: string;
    name: string;
    native: string;
    unified: string;
    keywords: string[];
    skins?: any[];
  }

  export interface EmojiPickerProps {
    native?: boolean;
    set?: string;
    skin?: number;
    emojiSize?: number;
    emojiTooltip?: boolean;
    title?: string;
    perLine?: number;
    emojiButtonSize?: number;
    emojiButtonColors?: {
      label?: string;
      text?: string;
    };
    categories?: {
      id: string;
      name: string;
      emojis: string[];
    }[];
    custom?: EmojiData[];
    autoFocus?: boolean;
    hideSearch?: boolean;
    hideGroupIcons?: boolean;
    hideObsolete?: boolean;
    hideEmojis?: string[];
    emojiVersion?: number;
    categoriesIcons?: any;
    categoryIcons?: any;
    groupIcons?: any;
    groupNames?: any;
    emojiGroupIcons?: any;
  }

  export const Picker: DefineComponent<EmojiPickerProps>;
  export const Emoji: DefineComponent<any>;
  export const NimbleEmoji: DefineComponent<any>;
  export const PickerProps: EmojiPickerProps;
  export const EmojiProps: any;
  export const NimbleEmojiProps: any;
}

declare module 'emoji-mart-vue-fast/src' {
  import { DefineComponent } from 'vue';

  export interface EmojiData {
    id: string;
    name: string;
    native: string;
    unified: string;
    keywords: string[];
    skins?: any[];
  }

  export interface EmojiPickerProps {
    native?: boolean;
    set?: string;
    skin?: number;
    emojiSize?: number;
    emojiTooltip?: boolean;
    title?: string;
    perLine?: number;
    emojiButtonSize?: number;
    emojiButtonColors?: {
      label?: string;
      text?: string;
    };
    categories?: {
      id: string;
      name: string;
      emojis: string[];
    }[];
    custom?: EmojiData[];
    autoFocus?: boolean;
    hideSearch?: boolean;
    hideGroupIcons?: boolean;
    hideObsolete?: boolean;
    hideEmojis?: string[];
    emojiVersion?: number;
    categoriesIcons?: any;
    categoryIcons?: any;
    groupIcons?: any;
    groupNames?: any;
    emojiGroupIcons?: any;
  }

  export const Picker: DefineComponent<EmojiPickerProps>;
  export const Emoji: DefineComponent<any>;
  export const NimbleEmoji: DefineComponent<any>;
  export const PickerProps: EmojiPickerProps;
  export const EmojiProps: any;
  export const NimbleEmojiProps: any;
  export class EmojiIndex {
    constructor(data: any);
    categories(): any[];
    emoji(emoji: string): EmojiData | null;
    search(query: string): EmojiData[];
  }
}
