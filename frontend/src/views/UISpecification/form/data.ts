import { FormSchema } from '@/components/ZFrom/src/types/form';
import { departmentDictionary } from '@/api/system/dept';

export const schemas: FormSchema[] = [
  {
    field: 'field1',
    component: 'Input',
    label: '字段1',
    colProps: {
      span: 8,
    },
    componentProps: {
      placeholder: '自定义placeholder',
      style: {
        width: '200px',
      },
      onChange: (e: any) => {
        console.log(e);
      },
    },
  },
  {
    field: 'field2',
    component: 'Select',
    label: '字段2',
    colProps: {
      span: 8,
    },
    componentProps: {
      placeholder: '自定义placeholder',
      style: {
        width: '200px',
      },
      options: [
        {
          label: '正常',
          value: '0',
        },
        {
          label: '禁用',
          value: '1',
        },
      ],
      onChange: (e: any) => {
        console.log(e);
      },
    },
  },
  {
    field: 'field3',
    component: 'Select',
    label: '字段2',
    colProps: {
      span: 8,
    },
    componentProps: {
      placeholder: '自定义placeholder',
      api: departmentDictionary,
      labelField: 'deptName',
      valueField: 'code',
      style: {
        width: '200px',
      },
      onChange: (e: any) => {
        console.log(e);
      },
    },
  },
  {
    field: 'field4',
    component: 'DatePicker',
    label: '时间选择',
    colProps: {
      span: 8,
    },
    componentProps: {
      placeholder: '请选择时间',
      style: {
        width: '200px',
      },
      type: 'date',
      onChange: (e: any) => {
        console.log(e);
      },
    },
  },
  {
    field: 'field5',
    component: 'DatePicker',
    label: '时间范围',
    colProps: {
      span: 8,
    },
    componentProps: {
      placeholder: '请选择时间范围​',
      startPlaceholder: '请选择开始时间​',
      endPlaceholder: '请选择结束时间​',
      type: 'daterange',
      onChange: (e: any) => {
        console.log(e);
      },
    },
  },
];
