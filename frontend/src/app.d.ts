declare module '*.vue' {
    import { defineComponent } from 'vue';
    const Component: ReturnType<typeof defineComponent>;
    export default Component;
}

declare type ComponentRef<T> = InstanceType<T>;

declare type ElRef<T extends HTMLElement = HTMLDivElement> = Nullable<T>

declare type LocaleType = 'zh-CN' | 'en'

declare type Recordable<T = any, K = string> = Record<K extends null | undefined ? string : K, T>

declare interface IResponse<T = any> {
  code: string
  data: T extends any ? T : T & any
}

declare type Nullable<T> = T | null
