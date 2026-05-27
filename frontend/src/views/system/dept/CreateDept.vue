<template>
  <Dialog v-model="dialogVisible" :title="title" width="600px" @close="onClose">
      <BasicForm ref="dialogFormRef" :rules="rules" @register="register"/>
      <template #footer>
        <ElButton @click="dialogVisible = false">关闭</ElButton>
        <ElButton type="primary" :loading="loading" @click="formSubmit">确定</ElButton>
      </template>
    </Dialog>
</template>

<script setup lang="ts">
import { getCurrentInstance, ComponentInternalInstance, ref, unref, reactive, watch, toRefs } from 'vue';
import { BasicForm, FormExpose } from '@/components/Form';
import { Dialog } from '@/components/Dialog';
import { deptSchemas } from './dept.config';
import { useForm } from '@/hooks/web/useForm';
import {  ElMessage } from 'element-plus';
import { listDept, addDept, updateDept, listDeptExcludeChild } from "@/api/system/dept";
const { proxy } = getCurrentInstance() as ComponentInternalInstance;

const emit = defineEmits(['submit', 'update:visible']);
defineExpose({name:'CreateUser'});
const props = defineProps({
  visible: {
    type: Boolean,
    default: false
  },
  title: {
    type: String,
    default:'添加用户'
  },
  deptData: {
    type: Object,
    default: () => ({})
  }
});
const {visible, title, deptData} = toRefs(props);
const dialogVisible = ref(false);
const dialogFormRef = ref<typeof BasicForm & FormExpose>();
const loading = ref(false);
const rules = reactive({
  parentId: [
    { required: true, message: "上级部门不能为空", trigger: "blur" },
  ],
  deptName: [
    { required: true, message: "部门名称不能为空", trigger: "blur" },
  ],
  orderNum: [
    { required: true, message: "显示排序不能为空", trigger: "blur" },
  ],
  email: [
    {
      type: "email",
      message: "请输入正确的邮箱地址",
      trigger: ["blur", "change"],
    },
  ],
  phone: [
    {
      pattern: /^1[3|4|5|6|7|8|9][0-9]\d{8}$/,
      message: "请输入正确的手机号码",
      trigger: "blur",
    },
  ],
});

const { register, methods } = useForm({
  schema: deptSchemas,
});

watch(visible, async (val)=>{
  dialogVisible.value = val as unknown as boolean;
});

watch(deptData, async (data) =>{
  if(data.isCreate){
    deptSchemas.forEach(el=>{
      if(el.field === 'parentId'){
        el.api = listDept;
        el.afterFetch = (deptData: any) => {
          return proxy!.handleTree(deptData.data, "deptId");
        };
      }
    });
    methods.setProps({
      schema:deptSchemas
    });
  } else {
    deptSchemas.forEach(el=>{
      if(el.field === 'parentId'){
        el.api = listDeptExcludeChild(data.deptId) as any;
        el.afterFetch = (deptData: any) => {
          return proxy!.handleTree(deptData.data, "deptId");
        };
      }
    });
    methods.setProps({
      schema:deptSchemas
    });
  }
  methods.setValues(data);
});


const onClose = () => {
  console.log(unref(dialogFormRef));
  emit('update:visible', false);
};

const formSubmit = async () =>{
  const elFormRef = unref(dialogFormRef)?.getElFormRef();
  await elFormRef?.validate(async (isValid: boolean) => {
    if (isValid) {
      loading.value = true;
      const formData = Object.assign(deptData.value, unref(dialogFormRef)?.formModel);
      if(formData?.deptId !== undefined){
        const res = await updateDept(formData) as any;
        if (res.code === 200) {
          emit('submit');
          ElMessage.success('修改成功');
        }
      } else {
        const res = await addDept(formData) as any;
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
