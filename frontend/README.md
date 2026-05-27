<p align="center">
	<img alt="logo" src="./src/assets/logo/logo.png">
</p>
<h1 align="center" style="margin: 30px 0 30px; font-weight: bold;">智先生 Vue3 + ts 版UI组件库</h1>
<h4 align="center">基于RuoYi v3.8.5 的 Vue3前端UI组件库</h4>

## 说明
智先生 Vue3 + ts 版UI组件库

fork 于 https://github.com/zzh948498/RuoYi-Vue3-ts.git
## 平台简介

* 本仓库为前端技术栈 [Vue3](https://v3.cn.vuejs.org) + [Element Plus](https://element-plus.org/zh-CN) + [Vite](https://cn.vitejs.dev) 版本。
* 配套后端代码仓库地址[RuoYi-Vue](https://gitee.com/y_project/RuoYi-Vue) 或 [RuoYi-Vue-fast](https://github.com/yangzongzhuan/RuoYi-Vue-fast) 版本。
* [RuoYi前端手册] (https://doc.ruoyi.vip/ruoyi-vue/document/qdsc.html#%E9%80%9A%E7%94%A8%E6%96%B9%E6%B3%95)

## 前端运行

拉取本项目 使用最先进的包管理工具pnpm
```bash
# 克隆项目
git clone https://10.10.200.23/MrHooge/zxs-admin-base.git

# 进入项目目录
cd zxs-admin-base

# 安装依赖
pnpm i

# 启动服务
pnpm dev

# 构建测试环境 yarn build:stage
# 构建生产环境 yarn build:prod
# 前端访问地址 http://localhost:80
```

## 内置功能

1.  用户管理：用户是系统操作者，该功能主要完成系统用户配置。
2.  部门管理：配置系统组织机构（公司、部门、小组），树结构展现支持数据权限。
3.  岗位管理：配置系统用户所属担任职务。
4.  菜单管理：配置系统菜单，操作权限，按钮权限标识等。
5.  角色管理：角色菜单权限分配、设置角色按机构进行数据范围权限划分。
6.  字典管理：对系统中经常使用的一些较为固定的数据进行维护。
7.  参数管理：对系统动态配置常用参数。
8.  通知公告：系统通知公告信息发布维护。
9.  操作日志：系统正常操作日志记录和查询；系统异常信息日志记录和查询。
10. 登录日志：系统登录日志记录查询包含登录异常。
11. 在线用户：当前系统中活跃用户状态监控。
12. 定时任务：在线（添加、修改、删除)任务调度包含执行结果日志。
13. 代码生成：前后端代码的生成（java、html、xml、sql）支持CRUD下载 。
14. 系统接口：根据业务代码自动生成相关的api接口文档。
15. 服务监控：监视当前系统CPU、内存、磁盘、堆栈等相关信息。
16. 缓存监控：对系统的缓存信息查询，命令统计等。
17. 在线构建器：拖动表单元素生成相应的HTML代码。
18. 连接池监视：监视当前系统数据库连接池状态，可进行分析SQL找出系统性能瓶颈。

## 在线体验

演示地址：http://vue.ruoyi.vip  
文档地址：http://doc.ruoyi.vip

