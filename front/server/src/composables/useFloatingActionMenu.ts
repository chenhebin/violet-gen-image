import { onBeforeUnmount, onMounted, ref } from 'vue'

const VIEWPORT_GAP = 12
const MENU_OFFSET = 4

export function useFloatingActionMenu(menuHeight: number) {
  const openMenuId = ref<string | null>(null)
  const menuPosition = ref({ top: 0, right: 0 })

  function closeMenu() {
    openMenuId.value = null
  }

  function toggleMenu(event: MouseEvent, itemId: string) {
    if (openMenuId.value === itemId) {
      closeMenu()
      return
    }

    const rect = (event.currentTarget as HTMLElement).getBoundingClientRect()
    menuPosition.value = {
      top:
        rect.bottom + menuHeight + VIEWPORT_GAP <= window.innerHeight
          ? rect.bottom + MENU_OFFSET
          : Math.max(VIEWPORT_GAP, rect.top - menuHeight - MENU_OFFSET),
      right: Math.max(VIEWPORT_GAP, window.innerWidth - rect.right),
    }
    openMenuId.value = itemId
  }

  onMounted(() => {
    window.addEventListener('resize', closeMenu)
    window.addEventListener('scroll', closeMenu, true)
  })

  onBeforeUnmount(() => {
    window.removeEventListener('resize', closeMenu)
    window.removeEventListener('scroll', closeMenu, true)
  })

  return { openMenuId, menuPosition, closeMenu, toggleMenu }
}
