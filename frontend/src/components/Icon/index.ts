import { defineComponent, h } from 'vue'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'

export const Icon = defineComponent({
  name: 'Icon',
  props: {
    icon: {
      type: String,
      required: true,
    },
    size: {
      type: Number,
      default: 14,
    },
    color: {
      type: String,
      default: '',
    },
  },
  setup(props) {
    return () => {
      if (props.icon && props.icon.startsWith('ep:')) {
        const iconName = props.icon.replace('ep:', '')
        const component = (ElementPlusIconsVue as Record<string, any>)[iconName]
        if (component) {
          return h(component, {
            style: {
              fontSize: `${props.size}px`,
              color: props.color || undefined,
            },
          })
        }
      }
      return h('i', {
        class: props.icon,
        style: {
          fontSize: `${props.size}px`,
          color: props.color || undefined,
        },
      })
    }
  },
})

export default Icon
