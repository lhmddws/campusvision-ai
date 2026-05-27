<template>
  <ContentWrap>
    <ElRow :gutter="20">
      <ElCol :span="4">
        <ElInput
            v-model="deptName"
            placeholder="请输入部门名称"
            clearable
            prefix-icon="Search"
            style="margin-bottom: 20px"
        />
        <div class="dept-tree">
          <ElTree
            ref="deptTreeRef"
            :data="deptOptions"
            :props="{ children: 'children', label:'label' }"
            :expand-on-click-node="false"
            :filter-node-method="filterNode"
            default-expand-all
            highlight-current
            @nodeClick="handleNodeClick"
          />
        </div>
      </ElCol>
      <ElCol :span="20">
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
          v-model:pageSize="tableObject.pageSize"
          v-model:currentPage="tableObject.currentPage"
          :columns="allSchemas.tableColumns"
          :data="tableObject.tableList"
          :loading="tableObject.loading"
          :pagination="{
            total: tableObject.total,
          }"
          @register="register"
          @selectChange="selectChange"
        >
          <template #status="{row}">
            <ElSwitch v-model="row.status" active-color="#02A797" active-value="0" inactive-value="1" @click="handleStatusChange(row)"/>
          </template>
          <template #action="{ row }">
            <ElButton
              v-hasPermi="['system:user:edit']"
              type="primary"
              text
              @click="handleUpdate(row)"
            >
              修改
            </ElButton>
            <ElButton
              v-hasPermi="['system:user:remove']"
              type="danger"
              text
              @click="handleDelete(row)"
            >
              删除
            </ElButton>
            <ElButton
              v-hasPermi="['system:user:resetPwd']"
              type="primary"
              text
              @click="handleResetPwd(row)"
            >
              重置密码
            </ElButton>
            <ElButton
              v-hasPermi="['system:user:edit']"
              type="primary"
              text
              @click="handleAuthRole(row)"
            >
              分配角色
            </ElButton>
          </template>
        </BasicTable>
      </ElCol>
    </ElRow>

    <!-- 添加用户 -->
    <CreateUser v-model:visible="showCreateUser" :userData="userData" :title="userTitle" @submit="getList"/>

    <!-- 导入弹窗 -->
    <el-dialog v-model="upload.open" :title="upload.title" width="400px" append-to-body>
      <FileUpload :limit="1" :uploadModel="upload"></FileUpload>
    </el-dialog>
  </ContentWrap>
</template>

<script setup lang="ts">
import { Search } from "@/components/Search";
import { ContentWrap } from "@/components/ContentWrap";
import { BasicTable } from "@/components/Table";
import { useTable } from "@/hooks/web/useTable";
import { useCrudSchemas } from "@/hooks/web/useCrudSchemas";
import { crudSchemas } from "./user.config";
import { getToken } from "@/utils/auth";
import {
  changeUserStatus,
  listUser,
  resetUserPwd,
  delUser,
} from "@/api/system/user";
import { treeSelect } from "@/api/system/dept";
import {
  getCurrentInstance,
  ComponentInternalInstance,
  ref,
  reactive,
  watch,
} from "vue";
import { useRouter } from "vue-router";
import { ElTree } from "element-plus";
import { downLoadExcel } from '@/utils/ruoyi';
import CreateUser from "./CreateUser.vue";

const router = useRouter();
const { proxy } = getCurrentInstance() as ComponentInternalInstance;
const showCreateUser = ref(false);
const { allSchemas } = useCrudSchemas(crudSchemas);
const deptName = ref('');
const deptOptions = ref<any[]>([]);
const userData = ref();
const isDisabled = ref(true);
const isUpdate = ref(true);
/*** 用户导入参数 */
const upload = ref({
  // 是否显示弹出层（用户导入）
  open: false,
  // 弹出层标题（用户导入）
  title: '用户导入',
  // 是否禁用上传
  isUploading: false,
  // 是否更新已经存在的用户数据
  updateSupport: 0,
  // 设置上传的请求头部
  headers: { Authorization: 'Bearer ' + getToken() },
  // 上传的地址
  uploadFileUrl: import.meta.env.VITE_APP_BASE_API + '/system/user/importData',
  // 模板下载信息
  template: {
    url: 'system/user/importTemplate',
    params: {},
    filename: ''
  }
});

const btns = reactive([
  {
    name: "新增",
    type: "primary",
    auth: ["system:user:add"],
    icon: "yijiantuisong",
    handler: () => {
      handleUpdate();
    }
  },
  {
    name: "修改",
    type: "primary",
    auth: ["system:user:edit"],
    icon: "yijianbanjie",
    disabled: isUpdate,
    handler: () => {
      handleUpdate();
    }
  },
  {
    name: "删除",
    type: "danger",
    auth: ["system:user:remove"],
    icon: "yijianbanjie",
    disabled: isDisabled,
    handler: () => {
      handleDelete();
    }
  },
  {
    name: "导入",
    auth: ["system:user:inport"],
    icon: "yijianbanjie",
    handler: ()=>{
      handleImport();
    }
  },
  {
    name: "导出",
    auth: ["system:user:export"],
    icon: "daochu",
    handler: ()=>{
      handleExport();
    }
  },
]);
const deptTreeRef = ref();
const searchRef = ref();
const userTitle = ref('添加用户');
const { register, tableObject, methods } = useTable({
  getListApi: listUser,
  delListApi: delUser as any,
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

/** 查询部门下拉树结构 */
const getTreeselect = async () => {
  const {data} = await treeSelect();
  deptOptions.value = data;
};
getTreeselect();
// 筛选节点
const filterNode = (value: any, data: any) =>{
  if (!value) return true;
  return data.label.indexOf(value) !== -1;
};
  // 节点单击事件
const handleNodeClick = (data: any) => {
  setSearchParams({deptId: data.id});
};

watch(deptName, (val)=>{
  console.log(val);
  deptTreeRef.value.filter(val);
});
/** 修改按钮操作 */
const handleUpdate = (row?: any) => {
  if(row){
    userTitle.value = '修改用户';
  } else {
    userTitle.value = '添加用户';
  }
  userData.value = row;
  showCreateUser.value = true;
};

/** 跳转角色分配 */
function handleAuthRole(row: any) {
  const userId = row.userId;
  router.push("/system/user-auth/role/" + userId);
}
/** 重置密码按钮操作 */
function handleResetPwd(row: any) {
  (proxy as any)
    .$prompt('请输入"' + row.userName + '"的新密码', "提示", {
      confirmButtonText: "确定",
      cancelButtonText: "取消",
      closeOnClickModal: false,
      inputPattern: /^.{5,20}$/,
      inputErrorMessage: "用户密码长度必须介于 5 和 20 之间",
    })
    .then(({ value }: any) => {
      resetUserPwd(row.userId, value).then((response) => {
        proxy!.$modal.msgSuccess("修改成功，新密码是：" + value);
      });
    })
    .catch((e: any) => {
      console.log(e);
    });
}

/** 用户状态修改  */
function handleStatusChange(row: any) {
  let text = row.status === "0" ? "启用" : "停用";
  proxy!.$modal
    .confirm('确认要"' + text + '""' + row.userName + '"用户吗?')
    .then(function () {
      return changeUserStatus(row.userId, row.status);
    })
    .then(() => {
      proxy!.$modal.msgSuccess(text + "成功");
    })
    .catch(function () {
      row.status = row.status === "0" ? "1" : "0";
    });
}

/** 删除按钮操作 */
async function handleDelete (row?: any) {
  const userIds = row?.userId ? [row?.userId] : (await getSelections()).map(d=>d.userId);
  console.log(userIds);

  proxy?.$modal
    .confirm('是否确认删除用户编号为"' + userIds + '"的数据项？')
    .then(function () {
      return delList(userIds, true);
    }).then(res=>{
      console.log(res);
    })
    .catch((e: any) => {
      console.log(e);
    });
}
/** 导出按钮操作 */
async function handleExport() {
  downLoadExcel(
    "system/user/export",
    {
      ...(await searchRef.value.methods.getFormData()),
    },
    `user_${new Date().getTime()}.xlsx`
  );
}
/** 导入按钮操作 */
const handleImport = () => {
  upload.value.open = true;
};
</script>

