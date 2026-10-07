import html2canvas from 'html2canvas'
import { brand, logos, activeLogo } from '@/config/brand'

// 课表导出图片：html2canvas 截取课表区域，叠加 logo + 品牌名平铺水印后下载 PNG。
// 水印仅在导出瞬间临时插入 DOM，截图完成即移除，不影响页面交互。

const logo = logos[activeLogo]

function watermarkSvg(text) {
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="260" height="160">
    <g opacity="0.08" transform="rotate(-24 130 80)">
      <text x="130" y="88" text-anchor="middle" font-size="22" font-weight="600"
        font-family="'PingFang SC','Microsoft YaHei',sans-serif" fill="#334155">${text}</text>
    </g>
  </svg>`
  return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`
}

/**
 * 导出指定元素为 PNG 图片（带品牌水印）。
 * @param {HTMLElement} el 要导出的 DOM 节点
 * @param {string} filename 下载文件名（不含扩展名）
 */
export async function exportElementToPng(el, filename = '我的课表') {
  if (!el) throw new Error('导出内容不存在')
  el.style.position = el.style.position || 'relative'

  // 平铺文字水印层 + 右下角 logo 署名层
  const mask = document.createElement('div')
  mask.style.cssText = [
    'position:absolute', 'inset:0', 'z-index:9', 'pointer-events:none',
    `background-image:url("${watermarkSvg(brand.brandName)}")`,
    'background-repeat:repeat'
  ].join(';')
  const sign = document.createElement('div')
  sign.style.cssText = [
    'position:absolute', 'right:16px', 'bottom:12px', 'z-index:10',
    'pointer-events:none', 'display:flex', 'align-items:center', 'gap:6px',
    'font-size:13px', 'font-weight:600', 'color:#64748b',
    "font-family:'PingFang SC','Microsoft YaHei',sans-serif"
  ].join(';')
  const img = document.createElement('img')
  img.src = logo.src
  img.style.cssText = 'width:18px;height:18px;object-fit:contain'
  sign.appendChild(img)
  sign.appendChild(document.createTextNode(brand.brandName))
  el.appendChild(mask)
  el.appendChild(sign)

  try {
    const canvas = await html2canvas(el, {
      scale: 2,
      useCORS: true,
      backgroundColor: '#ffffff',
      logging: false
    })
    const link = document.createElement('a')
    link.download = `${filename}-${new Date().toISOString().slice(0, 10)}.png`
    link.href = canvas.toDataURL('image/png')
    link.click()
  } finally {
    mask.remove()
    sign.remove()
  }
}
