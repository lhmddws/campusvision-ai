<template>
  <Dialog v-model="dialogVisible" :title="title" width="400px" @close="onClose">
      <BasicForm ref="dialogFormRef" @register="register">
        <template #dataScope>
          <el-select v-model="scopeFormData.dataScope" style="width: 100%" @change="dataScopeSelectChange">
            <el-option
              v-for="item in dataScopeOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            ></el-option>
          </el-select>
        </template>
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
            v-model="deptCheckStrictly"
            @change="handleCheckedTreeConnect($event)"
            >父子联动</el-checkbox
          >
          <el-tree
            ref="deptRef"
            class="tree-border"
            :data="deptOptions"
            show-checkbox
            node-key="id"
            :check-strictly="!deptCheckStrictly"
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
import {  ref, unref, watch, toRefs, nextTick } from 'vue';
import { BasicForm, FormExpose } from '@/components/Form';
import { Dialog } from '@/components/Dialog';
import { assignScopeSchemas } from './role.config';
import { useForm } from '@/hooks/web/useForm';
import {  ElMessage } from 'element-plus';
import {addRole,updateRole,deptTreeSelect} from "@/api/system/role";
import { FormSchema } from '@/types/form';

const emit = defineEmits(['submit', 'update:visible', 'close']);
defineExpose({name:'AssignPermissions'});
const props = defineProps({
  visible: {
    type: Boolean,
    default: false
  },
  title: {
    type: String,
    default:'分配数据权限'
  },
  roleData: {
    type: Object,
    default: () => ({})
  }
});
const {visible, title, roleData} = toRefs(props);
const deptOptions = ref<any[]>([]);
const dialogVisible = ref(false);
const dialogFormRef = ref<typeof BasicForm & FormExpose>();
const deptRef = ref<any>(null);
const scopeFormData = ref<any>({});
const loading = ref(false);
const menuExpand = ref(true);
const menuNodeAll = ref(false);
const deptCheckStrictly = ref(true);
const dataScopeOptions = ref([
  { value: "1", label: "全部数据权限" },
  { value: "2", label: "自定数据权限" },
  { value: "3", label: "本部门数据权限" },
  { value: "4", label: "本部门及以下数据权限" },
  { value: "5", label: "仅本人数据权限" },
]);

const { register, methods } = useForm({
  schema: assignScopeSchemas,
});


watch(visible, async (val)=>{
  dialogVisible.value = val as unknown as boolean;
});

watch(roleData, async (roleModel) =>{
  console.log(roleModel);
  scopeFormData.value = {...roleModel};
  if(scopeFormData.value.roleId !== undefined){
    await nextTick();
    const { data:roleMenuData } = await deptTreeSelect(scopeFormData.value.roleId) as any;
    deptOptions.value = roleMenuData.depts;
    roleMenuData.checkedKeys.forEach((v: any) => {
      nextTick(() => {
        deptRef.value.setChecked(v, true, false);
      });
    });
    if (scopeFormData.value.dataScope === "2") {
      handleCheckedTreeExpand(true);
    }
    assignScopeSchemas.forEach((el: FormSchema)=>{
      if(el.field === 'menuRole'){
        el.hidden = scopeFormData.value.dataScope !== "2";
      }
    });
    methods.setProps({
      schema:assignScopeSchemas
    });

    methods.setValues(scopeFormData.value);
  }
});


/** 树权限（展开/折叠）*/
async function handleCheckedTreeExpand(value: any) {
  for (let i = 0; i < deptOptions.value.length; i++) {
    await nextTick();
    deptRef.value.store.nodesMap[deptOptions.value[i].id].expanded = value;
  }
}
/** 树权限（全选/全不选） */
function handleCheckedTreeNodeAll(value: any) {
  deptRef.value.setCheckedNodes(value ? deptOptions.value : []);
}
/** 树权限（父子联动） */
function handleCheckedTreeConnect(value: any) {
  deptCheckStrictly.value = value ? true : false;
}

/** 所有菜单节点数据 */
function getMenuAllCheckedKeys() {
  // 目前被选中的菜单节点
  let checkedKeys = deptRef.value.getCheckedKeys();
  // 半选中的菜单节点
  let halfCheckedKeys = deptRef.value.getHalfCheckedKeys();
  checkedKeys.unshift.apply(checkedKeys, halfCheckedKeys);
  return checkedKeys;
}

/** 选择角色权限范围触发 */
function dataScopeSelectChange(value: any) {
  assignScopeSchemas.forEach((el: FormSchema)=>{
    if(el.field === 'menuRole'){
      el.hidden = value !== "2";
    }
  });
  methods.setProps({
    schema:assignScopeSchemas
  });
  if (value !== "2") {
    deptRef.value.setCheckedKeys([]);
  } else {
    handleCheckedTreeExpand(true);
  }
}

const onClose = () => {
  emit('close', false);
  emit('update:visible', false);
};

const formSubmit = async () =>{
  const elFormRef = unref(dialogFormRef)?.getElFormRef();
  await elFormRef?.validate(async (isValid: boolean) => {
    if (isValid) {
      loading.value = true;
      const formData = Object.assign(roleData.value, unref(dialogFormRef)?.formModel);
      formData.menuIds = getMenuAllCheckedKeys();
      formData.deptCheckStrictly = deptCheckStrictly.value;
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
