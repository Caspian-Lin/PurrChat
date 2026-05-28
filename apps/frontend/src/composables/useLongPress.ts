import { ref, onUnmounted } from 'vue';

interface Position {
  x: number;
  y: number;
}

/**
 * 长按检测 composable
 * 用于移动端长按触发上下文菜单
 * @param onLongPress 长按触发的回调函数
 * @param duration 长按触发时间（毫秒），默认 500ms
 * @param moveThreshold 移动取消阈值（像素），默认 10px
 */
export function useLongPress(
  onLongPress: (position: Position) => void,
  duration = 500,
  moveThreshold = 10
) {
  const isLongPressing = ref(false);
  let timer: ReturnType<typeof setTimeout> | null = null;
  let startPosition: Position | null = null;

  function onTouchStart(event: TouchEvent) {
    // 只处理单指触摸
    if (event.touches.length !== 1) return;

    const touch = event.touches[0];
    startPosition = { x: touch.clientX, y: touch.clientY };
    isLongPressing.value = false;

    timer = setTimeout(() => {
      if (startPosition) {
        isLongPressing.value = true;
        onLongPress(startPosition);
        // 触发轻微振动反馈（如果支持）
        if (navigator.vibrate) {
          navigator.vibrate(30);
        }
      }
    }, duration);
  }

  function onTouchMove(event: TouchEvent) {
    if (!startPosition || !timer) return;

    const touch = event.touches[0];
    const deltaX = Math.abs(touch.clientX - startPosition.x);
    const deltaY = Math.abs(touch.clientY - startPosition.y);

    // 移动超过阈值，取消长按
    if (deltaX > moveThreshold || deltaY > moveThreshold) {
      cancel();
    }
  }

  function onTouchEnd() {
    cancel();
  }

  function onTouchCancel() {
    cancel();
  }

  function cancel() {
    if (timer) {
      clearTimeout(timer);
      timer = null;
    }
    startPosition = null;
    isLongPressing.value = false;
  }

  onUnmounted(() => {
    cancel();
  });

  return {
    isLongPressing,
    handlers: {
      onTouchstart: onTouchStart,
      onTouchmove: onTouchMove,
      onTouchend: onTouchEnd,
      onTouchcancel: onTouchCancel,
    },
    cancel,
  };
}
