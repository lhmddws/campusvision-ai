import type { CSSProperties } from 'vue';
import { ColProps, ComponentProps, ComponentName } from '@/types/components';
import { FormValueType, FormValueType } from '@/types/form';
import type { AxiosPromise } from 'axios';

export type FormSetPropsType = {
  field: string
  path: string
  value: any
}
export interface Tools {
  name: string;
  type?: any;
  icon?: string;
  auth?: any;
  // eslint-disable-next-line @typescript-eslint/ban-types
  handler?: Function;
  disabled?: boolean | undefined;
}

export type FormValueType = string | number | string[] | number[] | boolean | undefined | null

export type FormItemProps = {
  labelWidth?: string | number
  required?: boolean
  rules?: Recordable
  error?: string
  showMessage?: boolean
  inlineMessage?: boolean
  style?: CSSProperties
}

export type FormSchema = {
  // 唯一值
  field: string
  // 标题
  label?: string
  // 提示
  labelMessage?: string
  // col组件属性
  colProps?: ColProps
  // 表单组件属性，slots对应的是表单组件的插槽，规则：${field}-xxx，具体可以查看element-plus文档
  componentProps?: { slots?: Recordable } & ComponentProps
  // formItem组件属性
  formItemProps?: FormItemProps
  // 渲染的组件
  component?: ComponentName
  // 初始值
  value?: FormValueType
  // 是否隐藏
  hidden?: boolean
  // 远程加载下拉项
  api?: <T = any>() => AxiosPromise<T>,
  // 获取远程数据之后对数据的操作并返回
  // eslint-disable-next-line @typescript-eslint/ban-types
  afterFetch?: Function,
  // 获取远程数据设置默认值
  // eslint-disable-next-line @typescript-eslint/ban-types
  apiValue?: string | Function
}
