import { useState, useEffect, useRef, useCallback } from 'react';

// Hook for intersection observer based animations
export const useIntersectionAnimation = (options?: IntersectionObserverInit) => {
  const [isVisible, setIsVisible] = useState(false);
  const [hasAnimated, setHasAnimated] = useState(false);
  const ref = useRef<HTMLElement>(null);

  useEffect(() => {
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting && !hasAnimated) {
          setIsVisible(true);
          setHasAnimated(true);
        }
      },
      { threshold: 0.1, ...options }
    );

    if (ref.current) {
      observer.observe(ref.current);
    }

    return () => observer.disconnect();
  }, [hasAnimated, options]);

  const reset = useCallback(() => {
    setIsVisible(false);
    setHasAnimated(false);
  }, []);

  return { ref, isVisible, hasAnimated, reset };
};

// Hook for spring animations
export const useSpring = (to: number, config?: { tension?: number; friction?: number; mass?: number }) => {
  const { tension = 120, friction = 14, mass = 1 } = config || {};
  const [value, setValue] = useState(to);
  const [velocity, setVelocity] = useState(0);
  const animationRef = useRef<number>();

  useEffect(() => {
    if (animationRef.current) {
      cancelAnimationFrame(animationRef.current);
    }

    const animate = () => {
      setValue(currentValue => {
        const force = -tension * (currentValue - to);
        const damping = -friction * velocity;
        const acceleration = (force + damping) / mass;
        
        const newVelocity = velocity + acceleration * 0.016; // 16ms frame time
        const newValue = currentValue + newVelocity * 0.016;
        
        setVelocity(newVelocity);
        
        // Stop animation when close enough and velocity is low
        if (Math.abs(newValue - to) < 0.01 && Math.abs(newVelocity) < 0.01) {
          return to;
        }
        
        animationRef.current = requestAnimationFrame(animate);
        return newValue;
      });
    };

    if (Math.abs(value - to) > 0.01) {
      animationRef.current = requestAnimationFrame(animate);
    }

    return () => {
      if (animationRef.current) {
        cancelAnimationFrame(animationRef.current);
      }
    };
  }, [to, tension, friction, mass, velocity]);

  return value;
};

// Hook for gesture animations (press, hover, etc.)
export const useGestureAnimation = (initialScale = 1) => {
  const [scale, setScale] = useState(initialScale);
  const [isPressed, setIsPressed] = useState(false);
  const [isHovered, setIsHovered] = useState(false);
  
  const animatedScale = useSpring(scale, { tension: 300, friction: 10 });

  const handlers = {
    onMouseDown: () => {
      setIsPressed(true);
      setScale(0.95);
    },
    onMouseUp: () => {
      setIsPressed(false);
      setScale(isHovered ? 1.05 : initialScale);
    },
    onMouseLeave: () => {
      setIsPressed(false);
      setIsHovered(false);
      setScale(initialScale);
    },
    onMouseEnter: () => {
      setIsHovered(true);
      if (!isPressed) {
        setScale(1.05);
      }
    },
    onTouchStart: () => {
      setIsPressed(true);
      setScale(0.95);
    },
    onTouchEnd: () => {
      setIsPressed(false);
      setScale(initialScale);
    }
  };

  return {
    scale: animatedScale,
    isPressed,
    isHovered,
    handlers
  };
};

// Hook for morphing animations (width, height, etc.)
export const useMorphAnimation = (
  from: { width?: number; height?: number; opacity?: number },
  to: { width?: number; height?: number; opacity?: number },
  trigger: boolean,
  duration = 300
) => {
  const [values, setValues] = useState(from);
  const animationRef = useRef<number>();
  const startTimeRef = useRef<number>();

  useEffect(() => {
    if (animationRef.current) {
      cancelAnimationFrame(animationRef.current);
    }

    startTimeRef.current = Date.now();
    const startValues = { ...values };
    const targetValues = trigger ? to : from;

    const animate = () => {
      const elapsed = Date.now() - (startTimeRef.current || 0);
      const progress = Math.min(elapsed / duration, 1);
      
      // Ease out cubic
      const easeProgress = 1 - Math.pow(1 - progress, 3);
      
      const newValues: typeof values = {};
      
      Object.keys(targetValues).forEach(key => {
        const typedKey = key as keyof typeof targetValues;
        const start = startValues[typedKey] || 0;
        const end = targetValues[typedKey] || 0;
        newValues[typedKey] = start + (end - start) * easeProgress;
      });
      
      setValues(newValues);
      
      if (progress < 1) {
        animationRef.current = requestAnimationFrame(animate);
      }
    };

    animationRef.current = requestAnimationFrame(animate);

    return () => {
      if (animationRef.current) {
        cancelAnimationFrame(animationRef.current);
      }
    };
  }, [trigger, duration, from, to]);

  return values;
};

// Hook for staggered animations
export const useStaggeredAnimation = (itemCount: number, staggerDelay = 100, trigger = true) => {
  const [animatedItems, setAnimatedItems] = useState<boolean[]>(new Array(itemCount).fill(false));

  useEffect(() => {
    if (!trigger) {
      setAnimatedItems(new Array(itemCount).fill(false));
      return;
    }

    const timeouts: NodeJS.Timeout[] = [];

    for (let i = 0; i < itemCount; i++) {
      const timeout = setTimeout(() => {
        setAnimatedItems(prev => {
          const newState = [...prev];
          newState[i] = true;
          return newState;
        });
      }, i * staggerDelay);
      
      timeouts.push(timeout);
    }

    return () => {
      timeouts.forEach(timeout => clearTimeout(timeout));
    };
  }, [itemCount, staggerDelay, trigger]);

  return animatedItems;
};

// Hook for page transition animations
export const usePageTransition = (isVisible: boolean, duration = 300) => {
  const [shouldRender, setShouldRender] = useState(isVisible);
  const [animationState, setAnimationState] = useState<'entering' | 'entered' | 'exiting' | 'exited'>(
    isVisible ? 'entered' : 'exited'
  );

  useEffect(() => {
    if (isVisible) {
      setShouldRender(true);
      setAnimationState('entering');
      const timeout = setTimeout(() => setAnimationState('entered'), 50);
      return () => clearTimeout(timeout);
    } else {
      setAnimationState('exiting');
      const timeout = setTimeout(() => {
        setAnimationState('exited');
        setShouldRender(false);
      }, duration);
      return () => clearTimeout(timeout);
    }
  }, [isVisible, duration]);

  return {
    shouldRender,
    animationState,
    isEntering: animationState === 'entering',
    isEntered: animationState === 'entered',
    isExiting: animationState === 'exiting',
    isExited: animationState === 'exited'
  };
};

// Hook for number counting animation
export const useCountAnimation = (
  target: number, 
  duration = 1000, 
  trigger = true,
  easing = 'easeOutQuart'
) => {
  const [count, setCount] = useState(0);
  const animationRef = useRef<number>();
  const startTimeRef = useRef<number>();

  const easingFunctions = {
    linear: (t: number) => t,
    easeOutQuart: (t: number) => 1 - Math.pow(1 - t, 4),
    easeOutCubic: (t: number) => 1 - Math.pow(1 - t, 3),
    easeOutCirc: (t: number) => Math.sqrt(1 - Math.pow(t - 1, 2))
  };

  useEffect(() => {
    if (!trigger) {
      setCount(0);
      return;
    }

    if (animationRef.current) {
      cancelAnimationFrame(animationRef.current);
    }

    startTimeRef.current = Date.now();
    const startValue = count;

    const animate = () => {
      const elapsed = Date.now() - (startTimeRef.current || 0);
      const progress = Math.min(elapsed / duration, 1);
      
      const easedProgress = easingFunctions[easing as keyof typeof easingFunctions](progress);
      const current = startValue + (target - startValue) * easedProgress;
      
      setCount(Math.round(current));
      
      if (progress < 1) {
        animationRef.current = requestAnimationFrame(animate);
      }
    };

    animationRef.current = requestAnimationFrame(animate);

    return () => {
      if (animationRef.current) {
        cancelAnimationFrame(animationRef.current);
      }
    };
  }, [target, duration, trigger, easing]);

  return count;
};