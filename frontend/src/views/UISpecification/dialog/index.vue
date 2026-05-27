<template>
  <ContentWrap>
    <ElButton type="primary" @click="dialogVisible = !dialogVisible">
      {{ t('dialogDemo.combineWithForm') }}
    </ElButton>

    <Dialog v-model="dialogVisible" :title="t('dialogDemo.dialog')" width="600px">
      <BasicForm ref="formRef" :schema="schema"/>
      <template #footer>
        <ElButton @click="dialogVisible = false">{{ t('dialogDemo.close') }}</ElButton>
        <ElButton type="primary" @click="formSubmit">{{ t('dialogDemo.submit') }}</ElButton>
      </template>
    </Dialog>
  </ContentWrap>
</template>

<script setup lang="ts">
import { ContentWrap } from '@/components/ContentWrap';
import { useI18n } from '@/hooks/web/useI18n';
import { ref, reactive, unref } from 'vue';
import { FormSchema } from '@/types/form';
import { BasicForm, FormExpose } from '@/components/Form';
import { Dialog } from '@/components/Dialog';
import { ElButton } from 'element-plus';


const { t } = useI18n();

const dialogVisible = ref(false);
const formRef = ref<FormExpose>();

const schema = reactive<FormSchema[]>([
  {
    field: 'field1',
    label: t('formDemo.input'),
    component: 'Input',
    colProps: {
      span: 12
    }
  },
  {
    field: 'field2',
    label: t('formDemo.select'),
    component: 'Select',
    colProps: {
      span: 12
    },
    componentProps: {
      style: {
        width: '100%'
      },
      options: [
        {
          label: 'option1',
          value: '1'
        },
        {
          label: 'option2',
          value: '2'
        }
      ]
    }
  },
]);

const formSubmit = () => {
  unref(formRef)
    ?.getElFormRef()
    ?.validate((valid) => {
      if (valid) {
        console.log('submit success');
      } else {
        console.log('submit fail');
      }
    });
};
</script>
