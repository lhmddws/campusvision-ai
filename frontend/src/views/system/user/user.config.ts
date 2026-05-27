import { CrudSchema } from '@/hooks/web/useCrudSchemas';
import { FormSchema } from '@/types/form';
import { treeSelect } from "@/api/system/dept";
import { getDicts } from '@/api/system/dict/data';
import { getUser } from "@/api/system/user";

import {  reactive } from 'vue';

export const crudSchemas = reactive<CrudSchema[]>([
  {
    field: 'userId',
    label: '用户编号',
    width: 120
  },
  {
    field: 'userName',
    label: '用户名称',
    search: {
      show: true,
      component: 'Input',
      componentProps: {
        style: {
          width: '200px'
        },
        placeholder: '请输入用户名称'
      }
    },
    width: 120
  },
  {
    field: 'phonenumber',
    label:  "手机号码",
    search: {
      show: true,
      component: 'Input',
      componentProps: {
        style: {
          width: '200px'
        },
        placeholder: '请输入手机号码'
      }
    },
    width: 120

  },
  {
    field: 'status',
    label: '用户状态',
    search: {
      show: true,
      component: 'Select',
      dictName: 'sys_normal_disable',
      componentProps: {
        placeholder: '请输入用户状态',
        options: []
      }
    },
    width: 120
  },
  {
    field: 'dept.deptName',
    label: '部门',
    minWidth: 220
  },
  {
    field: 'createTime',
    label: '创建时间',
    width: 220
  },
  {
    field: 'searchTime',
    label: '查询时间',
    search:{
      show: true,
      component: 'DatePicker',
      componentProps: {
        type: 'daterange',
        rangeSeparator:"-",
        startPlaceholder: '开始日期',
        endPlaceholder: '结束日期',
        valueFormat:"YYYY-MM-DD",
        style:{
          width: '240px'
        },
        trueNames: ['beginTime', 'endTime'],
        onChange: (val: any)=>{
          console.log(val);
        }
      }
    },
    table: {
      show: false
    },
  },
  {
    field: 'action',
    label: '操作',
    fixed: 'right',
    width: 280

  }
]);


export const userSchemas = reactive<FormSchema[]>([
  {
    field: 'nickName',
    label: '用户昵称',
    component: 'Input',
    colProps: {
      span: 12
    }
  },
  {
    field: 'deptId',
    label: '归属部门',
    component: 'TreeSelect',
    colProps: {
      span: 12
    },
    api: treeSelect,
    componentProps: {
      valueKey: 'id',
      style: {
        width: '100%'
      },
    },
  },
  {
    field: 'phonenumber',
    label: '手机号码',
    component: 'Input',
    colProps: {
      span: 12
    }
  },
  {
    field: 'email',
    label: '邮箱',
    component: 'Input',
    colProps: {
      span: 12
    }
  },
  {
    field: 'userName',
    label: '用户名称',
    component: 'Input',
    colProps: {
      span: 12
    }
  },
  {
    field: 'password',
    label: '用户密码',
    component: 'Input',
    colProps: {
      span: 12
    },
    componentProps: {
      disabled: false
    }
  },
  {
    field: 'sex',
    label: '用户性别',
    component: 'Select',
    api: getDicts('sys_user_sex'),
    colProps: {
      span: 12
    },
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
  {
    field: 'status',
    label: '状态',
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
  {
    field: 'postIds',
    label: '岗位',
    component: 'Select',
    api: getUser,
    afterFetch:(res: any)=>{
      return res.data.posts;
    },
    apiValue: 'postIds',
    colProps: {
      span: 12
    },
    componentProps: {
      multiple: true,
      optionsAlias:{
        labelField: 'postName',
        valueField: 'postId'
      },
      style: {
        width: '100%'
      }
    }
  },
  {
    field: 'roleIds',
    label: '角色',
    component: 'Select',
    api: getUser,
    afterFetch:(res: any)=>{
      return res.data.roles;
    },
    apiValue: 'roleIds',
    colProps: {
      span: 12
    },
    componentProps: {
      multiple: true,
      optionsAlias:{
        labelField: 'roleName',
        valueField: 'roleId'
      },
      style: {
        width: '100%'
      }
    }
  },
  {
    field: 'remark',
    label: '备注',
    component: 'Input',
    colProps: {
      span: 24
    },
    componentProps: {
      type:'textarea',
      placeholder: '请输入备注内容'
    }
  },
]);
