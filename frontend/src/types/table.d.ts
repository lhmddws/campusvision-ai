// 渲染函数，比如后端给你返回了0 代表 禁止， 1 代表启用
import type { VNode } from 'vue';

export interface TableColumnRenderer extends TableColumnScope {
  index: number;
  attrs: any;
}

export type TableColumn = {
  field: string
  label?: string
  // el-table fixed 属性 左右固定布局  只能在头尾的配置中存在
  fixed?: string;
  // el-table width 属性 宽度
  width?: number;
  minWidth?: string | number | undefined;
  //   属性 对其
  align?: string;
  // 自定义渲染
  render?: (data: TableColumnRenderer) => VNode;
  // 自定义slot
  slot?: string;
  //  行数据key
  rowKey?: string;
  //  是否默认开启所有行
  defaultExpandAll?: boolean;
  children?: TableColumn[]
} & Recordable

export type TableSlotDefault = {
  row: Recordable
  column: TableColumn
  $index: number
} & Recordable

export interface Pagination {
  small?: boolean
  background?: boolean
  pageSize?: number
  defaultPageSize?: number
  total?: number
  pageCount?: number
  pagerCount?: number
  currentPage?: number
  defaultCurrentPage?: number
  layout?: string
  pageSizes?: number[]
  popperClass?: string
  prevText?: string
  nextText?: string
  disabled?: boolean
  hideOnSinglePage?: boolean
}

export interface TableSetPropsType {
  field: string
  path: string
  value: any
}
