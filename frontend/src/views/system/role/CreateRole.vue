<template>
  <Dialog v-model="dialogVisible" :title="title" width="400px" @close="onClose">
      <BasicForm ref="dialogFormRef" :rules="rules" @register="register">
        <template #menuRole>
          <div>
            <el-checkbox
            v-model="menuExpand"
            @change="handleCheckedTreeExpand($event)"
            >展开/折叠</el-checkbox
          >
          <el-checkbox
            v-model="menuNodeAll"
            @change="handleCheckedTreeNodeAll($event)"
            >全选/全不选</el-checkbox
          >
          <el-checkbox
            v-model="menuCheckStrictly"
            @change="handleCheckedTreeConnect($event)"
            >父子联动</el-checkbox
          >
          <el-tree
            ref="menuRef"
            class="tree-border"
            :data="menuOptions"
            show-checkbox
            node-key="id"
            :check-strictly="!menuCheckStrictly"
            empty-text="加载中，请稍候"
            :props="{ label: 'label', children: 'children' }"
          ></el-tree>
          </div>
        </template>
      </BasicForm>
      <template #footer>
        <ElButton @click="dialogVisible = false">关闭</ElButton>
        <ElButton type="primary" :loading="loading" @click="formSubmit">确定</ElButton>
      </template>
    </Dialog>
</template>

<script setup lang="ts">
import { getCurrentInstance, ComponentInternalInstance, ref, unref, reactive, watch, toRefs, nextTick } from 'vue';
import { BasicForm, FormExpose } from '@/components/Form';
import { Dialog } from '@/components/Dialog';
import { roleSchemas } from './role.config';
import { useForm } from '@/hooks/web/useForm';
import {  ElMessage } from 'element-plus';
import {addRole,changeRoleStatus,dataScope,delRole,getRole,listRole,updateRole,deptTreeSelect} from "@/api/system/role";
import { roleMenuTreeselect, treeselect as menuTreeselect } from "@/api/system/menu";
const { proxy } = getCurrentInstance() as ComponentInternalInstance;

const emit = defineEmits(['submit', 'update:visible']);
defineExpose({name:'CreateRole'});
const props = defineProps({
  visible: {
    type: Boolean,
    default: false
  },
  title: {
    type: String,
    default:'添加用户'
  },
  roleData: {
    type: Object,
    default: () => ({})
  }
});
const {visible, title, roleData} = toRefs(props);
const menuOptions = ref<any[]>([]);
const dialogVisible = ref(false);
const dialogFormRef = ref<typeof BasicForm & FormExpose>();
const menuRef = ref<any>(null);
const loading = ref(false);
const menuExpand = ref(false);
const menuNodeAll = ref(false);
const menuCheckStrictly = ref(true);


const rules = reactive({
  roleName: [{ required: true, message: "角色名称不能为空", trigger: "blur" }],
  roleKey: [{ required: true, message: "权限字符不能为空", trigger: "blur" }],
  roleSort: [{ required: true, message: "角色顺序不能为空", trigger: "blur" }],
});

const { register, methods } = useForm({
  schema: roleSchemas,
});

watch(visible, async (val)=>{
  dialogVisible.value = val as unknown as boolean;
});

watch(roleData, async (roleModel) =>{
  if(roleModel.roleId !== undefined){
    const { data } = await getRole(roleModel.roleId);
    roleModel = data;
    roleModel.roleSort = Number(roleModel.roleSort);
    await nextTick();
    const { data:roleMenuData } = await roleMenuTreeselect(roleModel.roleId) as any;
    menuOptions.value = roleMenuData.menus;
    roleMenuData.checkedKeys.forEach((v: any) => {
      nextTick(() => {
        menuRef.value.setChecked(v, true, false);
      });
    });
    methods.setValues(roleModel);
  }
});

/** 查询菜单树结构 */
const getMenuTreeSelect= async() => {
  const { data } = await menuTreeselect();
  menuOptions.value = data;
};
getMenuTreeSelect();

/** 树权限（展开/折叠）*/
function handleCheckedTreeExpand(value: any) {
  let treeList = menuOptions.value;
  for (let i = 0; i < treeList.length; i++) {
    menuRef.value.store.nodesMap[treeList[i].id].expanded = value;
  }
}
/** 树权限（全选/全不选） */
function handleCheckedTreeNodeAll(value: any) {
  menuRef.value.setCheckedNodes(value ? menuOptions.value : []);
}
/** 树权限（父子联动） */
function handleCheckedTreeConnect(value: any) {
  menuCheckStrictly.value = value ? true : false;
}

/** 所有菜单节点数据 */
function getMenuAllCheckedKeys() {
  // 目前被选中的菜单节点
  let checkedKeys = menuRef.value.getCheckedKeys();
  // 半选中的菜单节点
  let halfCheckedKeys = menuRef.value.getHalfCheckedKeys();
  checkedKeys.unshift.apply(checkedKeys, halfCheckedKeys);
  return checkedKeys;
}

const onClose = () => {
  emit('update:visible', false);
};

const formSubmit = async () =>{
  const elFormRef = unref(dialogFormRef)?.getElFormRef();
  await elFormRef?.validate(async (isValid: boolean) => {
    if (isValid) {
      loading.value = true;
      const formData = Object.assign(roleData.value, unref(dialogFormRef)?.formModel);
      formData.menuIds = getMenuAllCheckedKeys();
      formData.menuCheckStrictly = menuCheckStrictly.value;
      if(formData?.roleId !== undefined){
        const res = await updateRole(formData) as any;
        if (res.code === 200) {
          emit('submit');
          ElMessage.success('修改成功');
        }
      } else {
        const res = await addRole(formData) as any;
        if (res.code === 200) {
          emit('submit');
          ElMessage.success('新增成功');
        }
      }
      loading.value = false;
      onClose();
    }
  });
};
</script>
