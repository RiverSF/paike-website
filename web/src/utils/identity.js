// 使用身份（userRole）展示文案集中在此维护。
// 师资身份设计已下线：所有账号统一按专职标准价计费，右上角与个人中心
// 展示的均为「使用身份」（老师 / 学员 / 机构）。

// 使用身份中文名：与注册表单「使用身份」选项、后端 model.UserRoleName 保持一致
// 注：家长（parent）与个人（personal）已融合为「学员」单一身份，统一显示为「学员」
export const USER_ROLE_NAME = {
  teacher: '老师',
  parent: '学员',
  personal: '学员',
  org: '机构'
}

// 使用身份中文名：优先使用后端下发的 userRoleName，避免前后端文案各自漂移
export function userRoleLabel(profile) {
  const role = profile?.userRole || 'teacher'
  // 融合身份：家长与个人统一显示为「学员」
  if (role === 'parent' || role === 'personal') return '学员'
  return profile?.userRoleName || USER_ROLE_NAME[role] || '老师'
}

// 兼容旧调用点：师资身份下线后，身份展示一律返回使用身份
export function identityLabel(profile) {
  return userRoleLabel(profile)
}
