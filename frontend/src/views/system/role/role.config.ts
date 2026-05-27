import { CrudSchema } from '@/hooks/web/useCrudSchemas';
import { FormSchema } from '@/types/form';
import { getDicts } from '@/api/system/dict/data';
import { ref, reactive } from 'vue';

const showScopeData = ref(false);

export const crudSchemas = reactive<CrudSchema[]>([
  {
    field: 'roleName',
    label: '角色名称',
    search: {
      show: true,
      component: 'Input',
      componentProps: {
        style: {
          width: '200px'
        },
        placeholder: '请输入角色名称'
      }
    },
  },
  {
    field: 'roleKey',
    label: '权限字符',
    search: {
      show: true,
      component: 'Input',
      componentProps: {
        style: {
          width: '200px'
        },
        placeholder: '请输入角色名称'
      }
    },
  },
  {
    field: 'roleSort',
    label: '显示顺序',
    width: 80
  },
  {
    field: 'status',
    label: '状态',
    search: {
      show: true,
      component: 'Select',
      dictName: 'sys_normal_disable',
      componentProps: {
        placeholder: '请选择角色状态',
        options: []
      }
    },
    width: 120
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
    field: 'createTime',
    label: '创建时间',
    width: 220
  },
  {
    field: 'action',
    label: '操作',
    fixed: 'right',
    width: 280
  }
]);

export const roleSchemas = reactive<FormSchema[]>([
  {
    field: 'roleName',
    label: '角色名称',
    component: 'Input',
    colProps: {
      span: 24
    },
    componentProps: {
      placeholder: '请输入角色名称',
    },
  },
  {
    field: 'roleKey',
    label: '权限字符',
    component: 'Input',
    labelMessage: '控制器中定义的权限字符，如：@PreAuthorize(`@ss.hasRole("admin")`)',
    colProps: {
      span: 24
    },
    componentProps: {
      placeholder: '请输入权限字符',
    },
  },
  {
    field: 'roleSort',
    label: '角色排序',
    component: 'InputNumber',
    colProps: {
      span: 24
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
    field: 'status',
    label: '状态',
    component: 'Radio',
    api: getDicts('sys_normal_disable'),
    colProps: {
      span: 24
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
    field: 'menuRole',
    label: '菜单权限',
    colProps: {
      span: 24
    },
  },
  {
    field: 'remark',
    label: '备注',
    component: 'Input',
    colProps: {
      span: 24
    },
    componentProps: {
      type: 'textarea',
      placeholder: '请输入备注信息'
    }
  },
]);

export const assignScopeSchemas = reactive<FormSchema[]>([

  {
    field: 'roleName',
    label: '角色名称',
    component: 'Input',
    colProps: {
      span: 24
    },
    componentProps: {
      disabled: true,
      placeholder: '请输入角色名称',
    },
  },
  {
    field: 'roleKey',
    label: '权限字符',
    component: 'Input',
    colProps: {
      span: 24
    },
    componentProps: {
      disabled: true,
      placeholder: '请输入权限字符',
    },
  },
  {
    field: 'dataScope',
    label: '权限范围',
    component: 'Select',
    colProps: {
      span: 24
    }
  },
  {
    field: 'menuRole',
    label: '数据权限',
    colProps: {
      span: 24
    },
    hidden: true
  },
]);

