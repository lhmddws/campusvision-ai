import request from '@/utils/request';

// 查询部门列表
export function listDept(query?: any) {
  return request({
    url: '/system/dept/list',
    method: 'get',
    params: query,
  });
}

// 查询部门列表（排除节点）
export function listDeptExcludeChild(deptId: any) {
  return request({
    url: '/system/dept/list/exclude/' + deptId,
    method: 'get',
  });
}

// 查询部门详细
export function getDept(deptId: any) {
  return request({
    url: '/system/dept/' + deptId,
    method: 'get',
  });
}
// 查询部门下拉树结构
export function treeSelect() {
  return request({
    url: "/system/dept/treeselect",
    method: "get",
  });
}

// 新增部门
export function addDept(data: any) {
  return request({
    url: '/system/dept',
    method: 'post',
    data: data,
  });
}

// 修改部门
export function updateDept(data: any) {
  return request({
    url: '/system/dept',
    method: 'put',
    data: data,
  });
}

// 删除部门
export function delDept(deptId: any) {
  return request({
    url: '/system/dept/' + deptId,
    method: 'delete',
  });
}

// 获取部门字典
export function departmentDictionary() {
  return request({
    url: "/api/sys/get/ks/name/dict/",
    method: "post"
  });
}
