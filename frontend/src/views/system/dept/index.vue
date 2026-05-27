<template>
  <ContentWrap>
    <Search
      ref="searchRef"
      :schema="allSchemas.searchSchema"
      layout="bottom"
      buttomPosition="left"
      :tools="btns"
      @search="setSearchParams"
      @reset="setSearchParams"
    />
    <BasicTable
      v-if="refreshTable"
      v-model:pageSize="tableObject.pageSize"
      v-model:currentPage="tableObject.currentPage"
      :columns="allSchemas.tableColumns"
      :data="tableObject.tableList"
      :loading="tableObject.loading"
      :pagination="{
        total: tableObject.total,
      }"
      row-key="deptId"
      :default-expand-all="isExpandAll"
      :selection="false"
      :tree-props="{ children: 'children', hasChildren: 'hasChildren' }"
      @register="register"
    >
      <template #status="{ row }">
        <dict-tag :options="sys_normal_disable" :value="row.status" />
      </template>
      <template #action="{ row }">
        <ElButton
          v-hasPermi="['system:user:resetPwd']"
          type="primary"
          text
          @click="handleUpdate(row, 'add')"
        >
          新增
        </ElButton>
        <ElButton
          v-hasPermi="['system:user:edit']"
          type="primary"
          text
          @click="handleUpdate(row,'edit')"
        >
          修改
        </ElButton>
        <ElButton
          v-if="row.parentId != 0"
          v-hasPermi="['system:user:remove']"
          type="danger"
          text
          @click="handleDelete(row)"
        >
          删除
        </ElButton>
      </template>
    </BasicTable>

    <!-- 添加部门 -->
    <CreateDept v-model:visible="showCreateDept" :deptData="deptData" :title="deptTitle" @submit="getList"/>

  </ContentWrap>
</template>

<script setup lang="ts">
import { Search } from "@/components/Search";
import { ContentWrap } from "@/components/ContentWrap";
import { BasicTable } from "@/components/Table";
import { useTable } from "@/hooks/web/useTable";
import { useCrudSchemas } from "@/hooks/web/useCrudSchemas";
import { crudSchemas } from "./dept.config";
import CreateDept from './CreateDept.vue';
import { getCurrentInstance, ComponentInternalInstance, ref, reactive, nextTick} from "vue";
import { listDept, getDept, delDept} from "@/api/system/dept";

const { allSchemas } = useCrudSchemas(crudSchemas);
const { proxy } = getCurrentInstance() as ComponentInternalInstance;

// eslint-disable-next-line camelcase
const { sys_normal_disable } = proxy!.useDict("sys_normal_disable");
const isExpandAll = ref(true);
const refreshTable = ref(true);
const deptTitle = ref('添加部门');
const deptData = ref();
const showCreateDept = ref(false);

const btns = reactive([
  {
    name: "新增",
    type: "primary",
    auth: ["system:dept:add"],
    icon: "xinzeng",
    handler: () => {
      handleUpdate();
    },
  },
  {
    name: "展开/折叠",
    type: "info",
    icon: "zhedie",
    handler: () => {
      handleTrigger();
    },
  },
]);

const { register, tableObject, methods } = useTable({
  getListApi: listDept as any,
  delListApi: delDept as any,
  response: {
    list: "data",
    total: "total",
  },
  afterFetch: (data: any) => {
    return proxy!.handleTree(data, "deptId");
  },
});
const { getList, setSearchParams, delList } = methods;
getList();

/** 修改按钮操作 */
const handleUpdate = async (row?: any, type?: string) => {
  if (type === 'edit') {
    deptTitle.value = "修改部门";
    const { data } = await getDept(row.deptId);
    data.orderNum = Number(data.orderNum);
    deptData.value = data;
  } else {
    deptTitle.value = "添加部门";
    if(row !== undefined){
      deptData.value = {parentId: row.deptId, isCreate: true};
    }
  }
  showCreateDept.value = true;
};

const handleTrigger = () => {
  refreshTable.value = false;
  isExpandAll.value = !isExpandAll.value;
  nextTick(() => {
    refreshTable.value = true;
  });
};


const handleDelete = (row?: any) => {
  proxy?.$modal
    .confirm('是否确认删除名称为"' + row.deptName + '"的数据项?')
    .then(function () {
      return delDept(row.deptId);
    })
    .then(() => {
      getList();
      proxy?.$modal.msgSuccess('删除成功');
    })
    .catch((e: any) => {
      console.log(e);
    });
};
</script>
