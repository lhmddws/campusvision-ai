<template>
  <ContentWrap>

    <BasicForm ref="formRef" :schema="schema" labelPosition="top" >

    </BasicForm>

  </ContentWrap>
</template>

<script setup lang="ts">
import { ContentWrap } from '@/components/ContentWrap';
import { BasicForm } from '@/components/Form';
import { useI18n } from '@/hooks/web/useI18n';
import { FormSchema } from '@/types/form';
import { useValidator } from '@/hooks/web/useValidator';
import { reactive, ref, unref } from 'vue';

const { required } = useValidator();
const { t } = useI18n();
const schema = reactive<FormSchema[]>([
  {
    field: 'field1',
    component: 'Input',
    label: '请输入内容',
    formItemProps: {
      rules: [required()],
    },
    componentProps: {
      style: {
        width: '200px'
      },
      placeholder: '请输入内容'
    }
  },
  {
    field: 'field2',
    component: 'Select',
    label: '请选择内容',
    componentProps: {
      options: [
        {
          label: 'option1',
          value: '1',
        },
        {
          label: 'option2',
          value: '2',
        },
      ],
      style: {
        width: '200px'
      },
      placeholder: '请输入内容',
      onChange: (value: string) => {
        console.log(value);
      },
    },
  },
  {
    field: 'field3',
    component: 'DatePicker',
    componentProps: {
      type: 'date',
      placeholder: t('formDemo.datePicker'),
    },
  },
  {
    field: 'field4',
    component: 'TimeSelect',
    componentProps: {
      placeholder: t('formDemo.timeSelect'),
    },
  },
  {
    field: 'field5',
    component: 'TreeSelect',
    componentProps: {
      placeholder: t('formDemo.treeSelect'),
      data:[
        {
          value: '1',
          label: 'Level one 1',
          children: [
            {
              value: '1-1',
              label: 'Level two 1-1',
              children: [
                {
                  value: '1-1-1',
                  label: 'Level three 1-1-1',
                },
              ],
            },
          ],
        },
        {
          value: '2',
          label: 'Level one 2',
          children: [
            {
              value: '2-1',
              label: 'Level two 2-1',
              children: [
                {
                  value: '2-1-1',
                  label: 'Level three 2-1-1',
                },
              ],
            },
            {
              value: '2-2',
              label: 'Level two 2-2',
              children: [
                {
                  value: '2-2-1',
                  label: 'Level three 2-2-1',
                },
              ],
            },
          ],
        },
      ]
    },
  },
]);


</script>
