const fun = function (evt: any) {
  let target = evt.target;
  if (target.nodeName === 'SPAN') {
    target = evt.target.parentNode;
  }
  target.blur();
};
export default {
  mounted(el: any, binding: any, vnode: any) {
    el.addEventListener('focus', fun);
  },
  unmounted(el: any) {
    el.removeEventListener('focus', fun);
  }
};
