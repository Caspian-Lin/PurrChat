/**
 * 节点布局工具 — Bot Studio 节点组件共享
 *
 * 提供 Handle 位置偏移计算等通用函数。
 */

// header(28px) + ports-padding(2px) + row_index * row_height(20px) + half_row(10px)
export function handleOffset(rowIndex: number, headerHeight = 30): string {
  return `${headerHeight + rowIndex * 20 + 10}px`;
}
