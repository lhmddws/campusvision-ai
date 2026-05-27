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
      row-key="roleId"
      @register="register"
      @selectChange="selectChange"
    >
      <template #status="{ row }">
        <ElSwitch
          v-model="row.status"
          active-color="#02A797"
          active-value="0"
          inactive-value="1"
          @click="handleStatusChange(row)"
        />
      </template>
      <template #action="{ row }">
        <ElButton
          v-hasPermi="['system:role:edit']"
          type="primary"
          text
          @click="handleUpdate(row)"
        >
          修改
        </ElButton>
        <ElButton
          v-if="row.parentId != 0"
          v-hasPermi="['system:role:remove']"
          type="danger"
          text
          @click="handleDelete(row)"
        >
          删除
        </ElButton>
        <ElButton
          v-hasPermi="['system:role:edit']"
          type="primary"
          text
          @click="handleDataScope(row)"
        >
          数据权限
        </ElButton>
        <ElButton
          v-hasPermi="['system:role:edit']"
          type="primary"
          text
          @click="handleAuthUser(row)"
        >
          分配用户
        </ElButton>
      </template>
    </BasicTable>

    <!-- 添加角色 -->
    <CreateRole v-model:visible="showCreateRole" :roleData="roleData" :title="roleTitle" @submit="getList"/>

    <!-- 数据权限 -->
    <AssignPermissions v-model:visible="showPermissions" :roleData="permData" title="分配数据权限" @submit="getList" @close="close"/>
  </ContentWrap>
</template>

<script setup name="Role" lang="ts">
import { Search } from "@/components/Search";
import { ContentWrap } from "@/components/ContentWrap";
import { BasicTable } from "@/components/Table";
import { useTable } from "@/hooks/web/useTable";
import { useCrudSchemas } from "@/hooks/web/useCrudSchemas";
import { crudSchemas } from "./role.config";
import CreateRole from "./CreateRole.vue";
import AssignPermissions from "./AssignPermissions.vue";
import {  getCurrentInstance,  ComponentInternalInstance,  ref,  reactive,} from "vue";
import { downLoadExcel } from '@/utils/ruoyi';
import {  delRole,  listRole,} from "@/api/system/role";
import { useRouter } from "vue-router";


const router = useRouter();

const { allSchemas } = useCrudSchemas(crudSchemas);
const { proxy } = getCurrentInstance() as ComponentInternalInstance;
const refreshTable = ref(true);
const roleTitle = ref("添加角色");
const roleData = ref();
const permData = ref();
const showCreateRole = ref(false);
const showPermissions = ref(false);
const isDisabled = ref(true);
const isUpdate = ref(true);
const searchRef = ref();

const btns = reactive([
  {
    name: "新增",
    type: "primary",
    auth: ["system:role:add"],
    icon: "xinzeng",
    handler: ()=> {
      handleUpdate();
    }
  },
  {
    name: "删除",
    type: "danger",
    auth: ["system:role:remove"],
    icon: "shanchu",
    disabled: isDisabled,
    handler: ()=> {
      handleDelete();
    }
  },
  {
    name: "导出",
    auth: ["system:role:export"],
    icon: "daochu",
    handler: ()=> {
      handleExport();
    }
  },
]);

const { register, tableObject, methods } = useTable({
  getListApi: listRole as any,
  delListApi: delRole as any,
  response: {
    list: "rows",
    total: "total",
  },
});
const { getList, setSearchParams, delList, getSelections } = methods;
getList();

const selectChange = (selected: any[])=>{
  isDisabled.value = selected.length < 1;
  isUpdate.value = selected.length !== 1;
};


const handleUpdate = (row?: any) => {
  if(row.roleId !== undefined){
    roleData.value = row;
    roleTitle.value = '修改角色';
  }
  showCreateRole.value = true;
};

const handleDelete = async (row?: any) => {
  const roleIds = row?.roleId ? [row?.roleId] : (await getSelections()).map(d=>d.roleId);
  proxy!.$modal
    .confirm('是否确认删除角色编号为"' + roleIds + '"的数据项?')
    .then(function () {
      return delList(roleIds, true);
    })
    .then(() => {
      getList();
      proxy!.$modal.msgSuccess("删除成功");
    })
    .catch((e: any) => {
      console.log(e);
    });
};
const handleDataScope = (row?: any) => {
  permData.value = row;
  showPermissions.value = true;
};
const close = () => {
  permData.value = {};
};
const handleAuthUser = (row?: any) => {
  console.log(row);
  router.push("/system/role-auth/user/" + row.roleId);

};
const handleStatusChange = (row?: any) => {
  console.log(row);
};
/** 导出按钮操作 */

async function handleExport() {
  downLoadExcel(
    "system/role/export",
    {
      ...(await searchRef.value.methods.getFormData()),
    },
    `role_${new Date().getTime()}.xlsx`
  );
}
</script>
