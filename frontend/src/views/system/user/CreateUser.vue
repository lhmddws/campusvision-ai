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
import { ref, unref, reactive, watch, toRefs } from 'vue';
import { BasicForm, FormExpose } from '@/components/Form';
import { Dialog } from '@/components/Dialog';
import { userSchemas } from './user.config';
import { useForm } from '@/hooks/web/useForm';
import { getConfigKey } from '@/api/system/config';
import {  ElMessage } from 'element-plus';
import { addUser, getUser, updateUser } from '@/api/system/user';

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
  userData: {
    type: Object,
    default: () => ({})
  }
});
const {visible, title, userData} = toRefs(props);
const dialogVisible = ref(false);
const dialogFormRef = ref<typeof BasicForm & FormExpose>();
const initPassword = ref('');
const loading = ref(false);
const rules = reactive({
  userName: [
    { required: true, message: "用户名称不能为空", trigger: "blur" },
    {
      min: 2,
      max: 20,
      message: "用户名称长度必须介于 2 和 20 之间",
      trigger: "blur",
    },
  ],
  nickName: [
    { required: true, message: "用户昵称不能为空", trigger: "blur" },
  ],
  password: [
    { required: true, message: "用户密码不能为空", trigger: "blur" },
    {
      min: 5,
      max: 20,
      message: "用户密码长度必须介于 5 和 20 之间",
      trigger: "blur",
    },
  ],
  email: [
    {
      type: "email",
      message: "请输入正确的邮箱地址",
      trigger: ["blur", "change"],
    },
  ],
  phonenumber: [
    { required: true, message: "手机号码不能为空", trigger: "blur" },
    {
      pattern: /^1[3|4|5|6|7|8|9][0-9]\d{8}$/,
      message: "请输入正确的手机号码",
      trigger: "blur",
    },
  ],
});

const { register, methods } = useForm({
  schema: userSchemas,
});

watch(visible, async (val)=>{
  dialogVisible.value = val as unknown as boolean;
  console.log(val);
});

const getInitPasswd = async () => {
  const {msg} = await getConfigKey("sys.user.initPassword") as any;
  initPassword.value = msg;
  userSchemas.forEach(el=>{
    if(el.field === 'password'){
      el.value = initPassword.value;
      if(el.componentProps){
        el.componentProps.disabled = true;
      }
    }
  });
};

watch(userData, async (data) =>{
  if(data.userId !== undefined){
    userSchemas.forEach(el=>{
      if(el.field === 'password' || el.field === 'userName' || el.field === 'password' || el.field === 'password'){
        el.hidden = true;
      }
      if(el.field === 'postIds' || el.field === 'roleIds'){
        el.api = getUser(data.userId) as any;
      }
    });
    methods.setProps({
      schema:userSchemas
    });
    methods.setValues(data);
  } else {
    getInitPasswd();
  }
});


const onClose = () => {
  emit('update:visible', false);
};

const formSubmit = async () =>{
  const elFormRef = unref(dialogFormRef)?.getElFormRef();
  await elFormRef?.validate(async (isValid: boolean) => {
    if (isValid) {
      loading.value = true;
      const formData = Object.assign(userData.value, unref(dialogFormRef)?.formModel);
      if(formData?.userId !== undefined){
        const res = await updateUser(formData) as any;
        if (res.code === 200) {
          emit('submit');
          ElMessage.success('修改成功');
        }
      } else {
        const res = await addUser(formData) as any;
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
