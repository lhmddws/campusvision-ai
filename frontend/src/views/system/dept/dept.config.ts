import { CrudSchema } from '@/hooks/web/useCrudSchemas';
import { FormSchema } from '@/types/form';
import { listDept } from "@/api/system/dept";
import { getDicts } from '@/api/system/dict/data';
import {  handleTree } from '@/utils/ruoyi';
import {  reactive } from 'vue';

export const crudSchemas = reactive<CrudSchema[]>([
  {
    field: 'deptName',
    label: '部门名称',
    search: {
      show: true,
      component: 'Input',
      componentProps: {
        style: {
          width: '200px'
        },
        placeholder: '请输入部门名称'
      }
    },
  },
  {
    field: 'orderNum',
    label: '排序',
    width: 80
  },
  {
    field: 'status',
    label: '部门状态',
    search: {
      show: true,
      component: 'Select',
      dictName: 'sys_normal_disable',
      componentProps: {
        placeholder: '请输入部门状态',
        options: []
      }
    },
    width: 120
  },
  {
    field: 'createTime',
    label: '创建时间',
    width: 220
  },
  {
    field: 'action',
    label: '操作',
    fixed: 'right',
    width: 180
  }
]);


export const deptSchemas = reactive<FormSchema[]>([
  {
    field: 'parentId',
    label: '上级部门',
    component: 'TreeSelect',
    colProps: {
      span: 24
    },
    api: listDept,
    afterFetch: (data: any) => {
      return handleTree(data.data, "deptId");
    },
    componentProps: {
      props: {
        label: 'deptName',
        value: 'deptId',
        children: 'children'
      },
      checkStrictly: true,
      style: {
        width: '100%'
      },
    },
  },
  {
    field: 'deptName',
    label: '部门名称',
    component: 'Input',
    colProps: {
      span: 12
    }
  },
  {
    field: 'orderNum',
    label: '显示排序',
    component: 'InputNumber',
    colProps: {
      span: 12
    },
    componentProps: {
      controlsPosition:"right",
      min: 0,
      style: {
        width: '100%'
      }
    },
    value: 0
  },
  {
    field: 'leader',
    label: '负责人',
    component: 'Input',
    colProps: {
      span: 12
    }
  },
  {
    field: 'phone',
    label: '联系电话',
    component: 'Input',
    colProps: {
      span: 12
    },
    componentProps: {
      maxlength: 11
    }
  },
  {
    field: 'email',
    label: '请输入邮箱',
    component: 'Input',
    colProps: {
      span: 12
    },
    componentProps: {
      maxlength: 50
    }
  },
  {
    field: 'status',
    label: '部门状态',
    component: 'Radio',
    api: getDicts('sys_normal_disable'),
    colProps: {
      span: 12
    },
    value: '0',
    componentProps: {
      optionsAlias: {
        labelField: 'dictLabel',
        valueField: 'dictValue',
      },
      style: {
        width: '100%'
      }
    }
  },
]);
