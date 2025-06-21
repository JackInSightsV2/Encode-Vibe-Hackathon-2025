import { useRef, useEffect, useState, useCallback } from 'react';

interface Point {
  x: number;
  y: number;
}

interface GestureState {
  isActive: boolean;
  startPoint: Point | null;
  currentPoint: Point | null;
  deltaX: number;
  deltaY: number;
  distance: number;
  angle: number;
  velocity: number;
  direction: 'up' | 'down' | 'left' | 'right' | null;
}

interface SwipeGestureOptions {
  threshold?: number;
  minVelocity?: number;
  maxTime?: number;
  onSwipeLeft?: () => void;
  onSwipeRight?: () => void;
  onSwipeUp?: () => void;
  onSwipeDown?: () => void;
  onSwipe?: (direction: 'up' | 'down' | 'left' | 'right', gesture: GestureState) => void;
}

export const useSwipeGesture = (options: SwipeGestureOptions = {}) => {
  const {
    threshold = 50,
    minVelocity = 0.3,
    maxTime = 500,
    onSwipeLeft,
    onSwipeRight,
    onSwipeUp,
    onSwipeDown,
    onSwipe
  } = options;

  const [gestureState, setGestureState] = useState<GestureState>({
    isActive: false,
    startPoint: null,
    currentPoint: null,
    deltaX: 0,
    deltaY: 0,
    distance: 0,
    angle: 0,
    velocity: 0,
    direction: null
  });

  const startTimeRef = useRef<number>(0);
  // const _rafRef = useRef<number>();

  const calculateGesture = useCallback((start: Point, current: Point, startTime: number) => {
    const deltaX = current.x - start.x;
    const deltaY = current.y - start.y;
    const distance = Math.sqrt(deltaX * deltaX + deltaY * deltaY);
    const angle = Math.atan2(deltaY, deltaX) * (180 / Math.PI);
    const time = Date.now() - startTime;
    const velocity = distance / Math.max(time, 1);

    let direction: 'up' | 'down' | 'left' | 'right' | null = null;
    if (Math.abs(deltaX) > Math.abs(deltaY)) {
      direction = deltaX > 0 ? 'right' : 'left';
    } else {
      direction = deltaY > 0 ? 'down' : 'up';
    }

    return {
      deltaX,
      deltaY,
      distance,
      angle,
      velocity,
      direction,
      time
    };
  }, []);

  const handleStart = useCallback((point: Point) => {
    startTimeRef.current = Date.now();
    setGestureState(prev => ({
      ...prev,
      isActive: true,
      startPoint: point,
      currentPoint: point,
      deltaX: 0,
      deltaY: 0,
      distance: 0,
      angle: 0,
      velocity: 0,
      direction: null
    }));
  }, []);

  const handleMove = useCallback((point: Point) => {
    setGestureState(prev => {
      if (!prev.isActive || !prev.startPoint) return prev;

      const gesture = calculateGesture(prev.startPoint, point, startTimeRef.current);
      
      return {
        ...prev,
        currentPoint: point,
        deltaX: gesture.deltaX,
        deltaY: gesture.deltaY,
        distance: gesture.distance,
        angle: gesture.angle,
        velocity: gesture.velocity,
        direction: gesture.direction
      };
    });
  }, [calculateGesture]);

  const handleEnd = useCallback(() => {
    setGestureState(prev => {
      if (!prev.isActive || !prev.startPoint || !prev.currentPoint) {
        return { ...prev, isActive: false };
      }

      const gesture = calculateGesture(prev.startPoint, prev.currentPoint, startTimeRef.current);
      const time = Date.now() - startTimeRef.current;

      // Check if gesture meets criteria
      if (
        gesture.distance >= threshold &&
        gesture.velocity >= minVelocity &&
        time <= maxTime
      ) {
        // Trigger appropriate callback
        switch (gesture.direction) {
          case 'left':
            onSwipeLeft?.();
            break;
          case 'right':
            onSwipeRight?.();
            break;
          case 'up':
            onSwipeUp?.();
            break;
          case 'down':
            onSwipeDown?.();
            break;
        }

        if (onSwipe && gesture.direction) {
          onSwipe(gesture.direction, { ...prev, ...gesture });
        }
      }

      return {
        isActive: false,
        startPoint: null,
        currentPoint: null,
        deltaX: 0,
        deltaY: 0,
        distance: 0,
        angle: 0,
        velocity: 0,
        direction: null
      };
    });
  }, [calculateGesture, threshold, minVelocity, maxTime, onSwipeLeft, onSwipeRight, onSwipeUp, onSwipeDown, onSwipe]);

  const handlers = {
    onTouchStart: (e: React.TouchEvent) => {
      const touch = e.touches[0];
      handleStart({ x: touch.clientX, y: touch.clientY });
    },
    onTouchMove: (e: React.TouchEvent) => {
      const touch = e.touches[0];
      handleMove({ x: touch.clientX, y: touch.clientY });
    },
    onTouchEnd: handleEnd,
    onMouseDown: (e: React.MouseEvent) => {
      handleStart({ x: e.clientX, y: e.clientY });
    },
    onMouseMove: (e: React.MouseEvent) => {
      if (gestureState.isActive) {
        handleMove({ x: e.clientX, y: e.clientY });
      }
    },
    onMouseUp: handleEnd,
    onMouseLeave: handleEnd
  };

  return {
    gestureState,
    handlers
  };
};

// Pinch gesture hook
interface PinchGestureOptions {
  threshold?: number;
  onPinchStart?: (scale: number) => void;
  onPinchMove?: (scale: number, delta: number) => void;
  onPinchEnd?: (scale: number) => void;
}

export const usePinchGesture = (options: PinchGestureOptions = {}) => {
  const { threshold: _threshold = 1.1, onPinchStart, onPinchMove, onPinchEnd } = options;
  
  const [gestureState, setGestureState] = useState({
    isActive: false,
    scale: 1,
    initialDistance: 0
  });

  const handleTouchStart = (e: React.TouchEvent) => {
    if (e.touches.length === 2) {
      const touch1 = e.touches[0];
      const touch2 = e.touches[1];
      const distance = Math.sqrt(
        Math.pow(touch2.clientX - touch1.clientX, 2) +
        Math.pow(touch2.clientY - touch1.clientY, 2)
      );

      setGestureState({
        isActive: true,
        scale: 1,
        initialDistance: distance
      });

      onPinchStart?.(1);
    }
  };

  const handleTouchMove = (e: React.TouchEvent) => {
    if (e.touches.length === 2 && gestureState.isActive) {
      const touch1 = e.touches[0];
      const touch2 = e.touches[1];
      const distance = Math.sqrt(
        Math.pow(touch2.clientX - touch1.clientX, 2) +
        Math.pow(touch2.clientY - touch1.clientY, 2)
      );

      const scale = distance / gestureState.initialDistance;
      const delta = scale - gestureState.scale;

      setGestureState(prev => ({ ...prev, scale }));
      onPinchMove?.(scale, delta);
    }
  };

  const handleTouchEnd = () => {
    if (gestureState.isActive) {
      onPinchEnd?.(gestureState.scale);
      setGestureState({
        isActive: false,
        scale: 1,
        initialDistance: 0
      });
    }
  };

  return {
    gestureState,
    handlers: {
      onTouchStart: handleTouchStart,
      onTouchMove: handleTouchMove,
      onTouchEnd: handleTouchEnd
    }
  };
};

// Long press gesture hook
interface LongPressOptions {
  delay?: number;
  threshold?: number;
  onLongPress?: (point: Point) => void;
  onLongPressStart?: (point: Point) => void;
  onLongPressEnd?: (point: Point) => void;
}

export const useLongPress = (options: LongPressOptions = {}) => {
  const { delay = 500, threshold = 10, onLongPress, onLongPressStart, onLongPressEnd } = options;
  
  const [state, setState] = useState({
    isActive: false,
    startPoint: null as Point | null
  });

  const timeoutRef = useRef<NodeJS.Timeout>();
  const startPointRef = useRef<Point | null>(null);

  const handleStart = useCallback((point: Point) => {
    startPointRef.current = point;
    setState({ isActive: true, startPoint: point });
    
    timeoutRef.current = setTimeout(() => {
      if (startPointRef.current) {
        onLongPressStart?.(startPointRef.current);
        onLongPress?.(startPointRef.current);
      }
    }, delay);
  }, [delay, onLongPress, onLongPressStart]);

  const handleMove = useCallback((point: Point) => {
    if (state.isActive && startPointRef.current) {
      const distance = Math.sqrt(
        Math.pow(point.x - startPointRef.current.x, 2) +
        Math.pow(point.y - startPointRef.current.y, 2)
      );

      if (distance > threshold) {
        // Movement too far, cancel long press
        if (timeoutRef.current) {
          clearTimeout(timeoutRef.current);
        }
        setState({ isActive: false, startPoint: null });
        startPointRef.current = null;
      }
    }
  }, [state.isActive, threshold]);

  const handleEnd = useCallback(() => {
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current);
    }
    
    if (state.isActive && startPointRef.current) {
      onLongPressEnd?.(startPointRef.current);
    }
    
    setState({ isActive: false, startPoint: null });
    startPointRef.current = null;
  }, [state.isActive, onLongPressEnd]);

  const handlers = {
    onTouchStart: (e: React.TouchEvent) => {
      const touch = e.touches[0];
      handleStart({ x: touch.clientX, y: touch.clientY });
    },
    onTouchMove: (e: React.TouchEvent) => {
      const touch = e.touches[0];
      handleMove({ x: touch.clientX, y: touch.clientY });
    },
    onTouchEnd: handleEnd,
    onMouseDown: (e: React.MouseEvent) => {
      handleStart({ x: e.clientX, y: e.clientY });
    },
    onMouseMove: (e: React.MouseEvent) => {
      handleMove({ x: e.clientX, y: e.clientY });
    },
    onMouseUp: handleEnd,
    onMouseLeave: handleEnd
  };

  useEffect(() => {
    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, []);

  return {
    state,
    handlers
  };
};

// Combined gesture hook
interface MultiGestureOptions extends SwipeGestureOptions, PinchGestureOptions, LongPressOptions {
  disabled?: boolean;
}

export const useMultiGesture = (options: MultiGestureOptions = {}) => {
  const { disabled = false, ...gestureOptions } = options;
  
  const swipe = useSwipeGesture(gestureOptions);
  const pinch = usePinchGesture(gestureOptions);
  const longPress = useLongPress(gestureOptions);

  const combinedHandlers = {
    onTouchStart: (e: React.TouchEvent) => {
      if (disabled) return;
      
      if (e.touches.length === 1) {
        swipe.handlers.onTouchStart(e);
        longPress.handlers.onTouchStart(e);
      } else if (e.touches.length === 2) {
        pinch.handlers.onTouchStart(e);
      }
    },
    onTouchMove: (e: React.TouchEvent) => {
      if (disabled) return;
      
      if (e.touches.length === 1) {
        swipe.handlers.onTouchMove(e);
        longPress.handlers.onTouchMove(e);
      } else if (e.touches.length === 2) {
        pinch.handlers.onTouchMove(e);
      }
    },
    onTouchEnd: (_e: React.TouchEvent) => {
      if (disabled) return;
      
      swipe.handlers.onTouchEnd();
      pinch.handlers.onTouchEnd();
      longPress.handlers.onTouchEnd();
    },
    onMouseDown: (e: React.MouseEvent) => {
      if (disabled) return;
      
      swipe.handlers.onMouseDown(e);
      longPress.handlers.onMouseDown(e);
    },
    onMouseMove: (e: React.MouseEvent) => {
      if (disabled) return;
      
      swipe.handlers.onMouseMove(e);
      longPress.handlers.onMouseMove(e);
    },
    onMouseUp: (_e: React.MouseEvent) => {
      if (disabled) return;
      
      swipe.handlers.onMouseUp();
      longPress.handlers.onMouseUp();
    },
    onMouseLeave: (_e: React.MouseEvent) => {
      if (disabled) return;
      
      swipe.handlers.onMouseLeave();
      longPress.handlers.onMouseLeave();
    }
  };

  return {
    swipe: swipe.gestureState,
    pinch: pinch.gestureState,
    longPress: longPress.state,
    handlers: combinedHandlers
  };
};

export default useSwipeGesture;