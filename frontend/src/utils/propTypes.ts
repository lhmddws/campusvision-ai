// import { createTypes, VueTypesInterface, VueTypeValidableDef } from 'vue-types';

// // 自定义扩展vue-types
// type PropTypes = VueTypesInterface & {
//   readonly style: VueTypeValidableDef<CSSProperties>
// }

// const propTypes = createTypes({
//   func: undefined,
//   bool: undefined,
//   string: undefined,
//   number: undefined,
//   object: undefined,
//   integer: undefined
// }) as PropTypes

// // 需要自定义扩展的类型
// // see: https://dwightjack.github.io/vue-types/advanced/extending-vue-types.html#the-extend-method
// propTypes.extend([
//   {
//     name: 'style',
//     getter: true,
//     type: [String, Object],
//     default: undefined
//   }
// ])

// export { propTypes }

// import
// - VueTypes library
// - validation object interface (VueTypeDef)
//   -  use VueTypeValidableDef if the new type is going to support the `validate` method.
// - the default VueType interface (VueTypesInterface)
import {
  VueTypeValidableDef,
  VueTypesInterface,
  createTypes
} from 'vue-types';
import { CSSProperties } from 'vue';

interface ProjectTypes extends VueTypesInterface {
  readonly style: VueTypeValidableDef<CSSProperties>
}


const propTypes = createTypes({
  func: undefined,
  bool: undefined,
  string: undefined,
  number: undefined,
  object: undefined,
  integer: undefined
}) as ProjectTypes;

propTypes.extend({
  name: 'style',
  getter: true,
  type: [String, Object],
  default: undefined
});

export default propTypes as ProjectTypes;
