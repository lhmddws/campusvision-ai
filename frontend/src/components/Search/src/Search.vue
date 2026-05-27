<script setup lang="ts">
import { BasicForm } from "@/components/Form";
import { PropType, computed, unref, ref } from "vue";
import VueTypes from '@/utils/propTypes';
import { ElButton } from "element-plus";
import { useI18n } from "@/hooks/web/useI18n";
import { useForm } from "@/hooks/web/useForm";
import { findIndex } from "@/utils";
import { cloneDeep } from "lodash-es";
import { FormSchema, Tools } from "@/types/form";
import { parseTime, addDateRange } from "@/utils/ruoyi";

const { t } = useI18n();

const props = defineProps({
  // 生成Form的布局结构数组
  schema: {
    type: Array as PropType<FormSchema[]>,
    default: () => [],
  },
  // 是否需要栅格布局
  isCol: VueTypes.bool.def(false),
  // 表单label宽度
  labelWidth: VueTypes.oneOfType([String, Number]).def("100px"),
  // 操作按钮风格位置
  layout: VueTypes.string
    .validate((v: string) => ["inline", "bottom"].includes(v))
    .def("bottom"),
  // 底部按钮的对齐方式
  buttomPosition: VueTypes.string
    .validate((v: string) => ["left", "center", "right"].includes(v))
    .def("left"),
  showSearch: VueTypes.bool.def(true),
  showReset: VueTypes.bool.def(true),
  // 是否显示伸缩
  expand: VueTypes.bool.def(false),
  // 伸缩的界限字段
  expandField: VueTypes.string.def(""),
  inline: VueTypes.bool.def(true),
  model: {
    type: Object as PropType<Recordable>,
    default: () => ({}),
  },
  // 操作栏
  tools: {
    type: Array as PropType<Tools[]>,
    default: () => [],
  },
  //  隐藏Label
  hiddenLabel: VueTypes.bool.def(true),
});

const emit = defineEmits(["search", "reset"]);

const visible = ref(true);

const newSchema = computed(() => {
  let schema: FormSchema[] = cloneDeep(props.schema);
  if (props.expand && props.expandField && !unref(visible)) {
    const index = findIndex(schema, (v: FormSchema) => v.field === props.expandField);
    if (index > -1) {
      const length = schema.length;
      schema.splice(index + 1, length);
    }
  }
  if (props.layout === "inline") {
    schema = schema.concat([
      {
        field: "action",
        formItemProps: {
          labelWidth: "100px",
        },
      },
    ]);
  }
  if (props.hiddenLabel) {
    schema.forEach((el) => {
      el.label = "";
    });
  }
  return schema;
});

const { register, elFormRef, methods } = useForm({
  model: props.model || {},
});

defineExpose({
  methods
});

const search = async () => {
  await unref(elFormRef)?.validate(async (isValid) => {
    if (isValid) {
      const dateRangerArr = props.schema.filter((d: FormSchema)=>d.component === 'DatePicker' && d?.componentProps.type === 'daterange');
      const { getFormData } = methods;
      let model = await getFormData();
      // let searchModel:any = {};
      dateRangerArr.forEach((el: any)=>{
        if(model && model[el.field]){
          model = {...addDateRange(model, model[el.field], el.componentProps.trueNames)};
          // delete searchModel[el.field];
        }
      });
      emit("search", model);
    }
  });
};

const reset = async () => {
  unref(elFormRef)?.resetFields();
  const { getFormData } = methods;
  const model: any = await getFormData();
  model.params = '';
  emit("reset", model);
};

const bottonButtonStyle = computed(() => {
  return {
    textAlign: (props.buttomPosition as unknown) as "left" | "center" | "right",
  };
});

const setVisible = () => {
  unref(elFormRef)?.resetFields();
  visible.value = !unref(visible);
};
</script>

<template>
  <BasicForm
    :is-custom="false"
    :label-width="labelWidth"
    hide-required-asterisk
    :inline="inline"
    :is-col="isCol"
    :schema="newSchema"
    @register="register"
  >
    <template #action>
      <div v-if="layout === 'inline'">
        <ElButton v-if="showSearch" v-btn type="primary" @click="search">
          <span class="iconfont mr-1 icon-sousuo" />
          {{ t("common.query") }}
        </ElButton>
        <ElButton v-if="showReset" v-btn @click="reset">
          <span class="iconfont mr-1 icon-zhongzhi" />
          {{ t("common.reset") }}
        </ElButton>
        <ElButton v-if="expand" v-btn text @click="setVisible">
          {{ t(visible ? "common.shrink" : "common.expand") }}
          <Icon :icon="visible ? 'ant-design:up-outlined' : 'ant-design:down-outlined'" />
        </ElButton>
      </div>
    </template>
  </BasicForm>
  <ElRow :gutter="16">
    <ElCol :span="8">
      <template v-if="layout === 'bottom'">
        <div :style="bottonButtonStyle">
          <ElButton v-if="showSearch" v-btn type="primary" @click="search">
            <span class="iconfont mr-1 icon-sousuo" />
            {{ t("common.query") }}
          </ElButton>
          <ElButton v-if="showReset" v-btn @click="reset">
            <span class="iconfont mr-1 icon-zhongzhi" />
            {{ t("common.reset") }}
          </ElButton>
          <ElButton v-if="expand" v-btn text @click="setVisible">
            {{ t(visible ? "common.shrink" : "common.expand") }}
            <Icon
              :icon="visible ? 'ant-design:up-outlined' : 'ant-design:down-outlined'"
            />
          </ElButton>
        </div>
      </template>
    </ElCol>
    <ElCol :span="16" style="text-align: right">
      <ElButton
        v-for="(item, index) in tools"
        :key="index"
        v-btn
        :type="item.type || 'default'"
        :v-hasPermi="item.auth"
        :disabled="item.disabled"
        plain
        @click="item.handler"
      >
        <span class="iconfont mr-1" :class="'icon-' + item.icon" />
        <span>
          {{ item.name }}
        </span>
      </ElButton>
    </ElCol>
  </ElRow>
</template>
