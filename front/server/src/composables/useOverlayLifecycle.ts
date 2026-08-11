import type { Ref } from 'vue'
import { onBeforeUnmount, watch } from 'vue'

let nextOverlayId = 1
const openOverlayIds: number[] = []

function setOverlayState(id: number, isOpen: boolean): void {
  const index = openOverlayIds.indexOf(id)
  if (isOpen && index === -1) openOverlayIds.push(id)
  if (!isOpen && index >= 0) openOverlayIds.splice(index, 1)
  document.body.classList.toggle('is-dialog-open', openOverlayIds.length > 0)
}

export function useOverlayLifecycle(
  isOpen: Ref<boolean>,
  onClose: () => void,
): void {
  const overlayId = nextOverlayId++

  const onKeydown = (event: KeyboardEvent): void => {
    if (
      event.key === 'Escape' &&
      isOpen.value &&
      openOverlayIds.at(-1) === overlayId
    ) {
      onClose()
    }
  }

  watch(
    isOpen,
    (next) => setOverlayState(overlayId, next),
    { immediate: true },
  )

  document.addEventListener('keydown', onKeydown)

  onBeforeUnmount(() => {
    document.removeEventListener('keydown', onKeydown)
    setOverlayState(overlayId, false)
  })
}
