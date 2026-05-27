<template>
  <ContentWrap>
    <Search
      :schema="allSchemas.searchSchema"
      layout="bottom"
      buttomPosition="left"
      :tools="btns"
      @search="setSearchParams"
      @reset="setSearchParams"
    />
    <BasicTable
      v-model:pageSize="tableObject.pageSize"
      v-model:currentPage="tableObject.currentPage"
      :columns="allSchemas.tableColumns"
      :data="tableObject.tableList"
      :loading="tableObject.loading"
      :pagination="{
        total: tableObject.total
      }"
      @register="register"
    >
      <template #action>
        <ElButton type="primary" text>
          {{ t('exampleDemo.edit') }}
        </ElButton>
        <ElButton type="primary" text>
          {{ t('exampleDemo.detail') }}
        </ElButton>
        <ElButton type="danger" text>
          {{ t('exampleDemo.del') }}
        </ElButton>
      </template>
    </BasicTable>
  </ContentWrap>
</template>

<script setup lang="ts">
import { Search } from '@/components/Search';
import { ContentWrap } from '@/components/ContentWrap';
import { BasicTable } from '@/components/Table';
import { reactive } from 'vue';
import { useI18n } from '@/hooks/web/useI18n';
import { useTable } from '@/hooks/web/useTable';
import { CrudSchema, useCrudSchemas } from '@/hooks/web/useCrudSchemas';
import {listUser} from '@/api/system/user';
import { listDept } from '@/api/system/dept';

export type TableData = {
  id: string
  author: string
  title: string
  content: string
  importance: number
  display_time: string
  pageviews: number
}
const { t } = useI18n();

const crudSchemas = reactive<CrudSchema[]>([
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
        placeholder: '请输入用户名称'
      }
    },
    width: 120
  },
  {
    field: 'nickName',
    label:  "用户昵称",
    search: {
      show: true,
      component: 'Input',
      componentProps: {
        placeholder: '请输入用户昵称'
      }
    },
    width: 120

  },
  {
    field: 'deptName',
    label: '部门',
    search: {
      show: true,
      api: listDept,
      component: 'Select',
      componentProps: {
        placeholder: '请输入部门',
        optionsAlias: {
          labelField: 'deptName'
        }
      }
    },
    width: 220
  },
  {
    field: 'deptName',
    label: '部门',
    width: 220
  },
  {
    field: 'deptName',
    label: '部门',
    width: 220
  },
  {
    field: 'deptName',
    label: '部门',
    width: 220
  },
  {
    field: 'deptName',
    label: '部门',
    width: 220
  },

  {
    field: 'createTime',
    label: '创建时间',
    width: 220
  },
  {
    field: 'action',
    label: t('tableDemo.action'),
    fixed: 'right',
    width: 180

  }
]);

const { allSchemas } = useCrudSchemas(crudSchemas);

const btns = reactive([
  {
    name: '一键推送',
    type: 'primary',
    auth: ['business:onConfirm:push'],
    icon: 'yijiantuisong',

  },
  {
    name: '一键办结',
    type: 'primary',
    auth: ['business:onConfirm:finish'],
    icon: 'yijianbanjie',

  },
  {
    name: '导出',
    auth: ['business:onConfirm:export'],
    icon: 'daochu',

  },
]);


const { register, tableObject, methods } = useTable<TableData>({
  getListApi: listUser,
  response: {
    list: 'rows',
    total: 'total'
  }
});

const { getList, setSearchParams } = methods;
getList();

</script>

