import { useState, useRef, useCallback } from 'react';

interface DragAndDropOptions<T> {
  onDrop: (draggedItem: T, targetItem: T, position: 'before' | 'after' | 'inside') => void;
  onReorder?: (items: T[]) => void;
  enabled?: boolean;
  allowNesting?: boolean;
  dragPreview?: (item: T) => React.ReactNode;
  dropEffect?: 'move' | 'copy' | 'link';
}

interface DragState<T> {
  draggedItem: T | null;
  dragOverItem: T | null;
  dropPosition: 'before' | 'after' | 'inside' | null;
  isDragging: boolean;
  dragPreviewElement: HTMLElement | null;
}

export const useDragAndDrop = <T extends { id: string }>(
  items: T[],
  options: DragAndDropOptions<T>
) => {
  const [dragState, setDragState] = useState<DragState<T>>({
    draggedItem: null,
    dragOverItem: null,
    dropPosition: null,
    isDragging: false,
    dragPreviewElement: null
  });

  const dragPreviewRef = useRef<HTMLDivElement>(null);

  const getDragHandlers = useCallback((item: T) => {
    const handleDragStart = (event: React.DragEvent) => {
      if (!options.enabled) return;
      
      setDragState(prev => ({
        ...prev,
        draggedItem: item,
        isDragging: true
      }));

      // Set drag data
      event.dataTransfer.effectAllowed = options.dropEffect || 'move';
      event.dataTransfer.setData('application/json', JSON.stringify(item));

      // Create custom drag preview if provided
      if (options.dragPreview && dragPreviewRef.current) {
        const previewElement = dragPreviewRef.current;
        previewElement.style.position = 'absolute';
        previewElement.style.top = '-1000px';
        document.body.appendChild(previewElement);
        event.dataTransfer.setDragImage(previewElement, 0, 0);
        
        setDragState(prev => ({
          ...prev,
          dragPreviewElement: previewElement
        }));
      }
    };

    const handleDragEnd = () => {
      setDragState(prev => {
        // Clean up drag preview
        if (prev.dragPreviewElement && document.body.contains(prev.dragPreviewElement)) {
          document.body.removeChild(prev.dragPreviewElement);
        }
        
        return {
          draggedItem: null,
          dragOverItem: null,
          dropPosition: null,
          isDragging: false,
          dragPreviewElement: null
        };
      });
    };

    return {
      draggable: options.enabled,
      onDragStart: handleDragStart,
      onDragEnd: handleDragEnd,
      'data-drag-id': item.id
    };
  }, [options.enabled, options.dropEffect, options.dragPreview]);

  const getDropHandlers = useCallback((item: T) => {
    const handleDragOver = (event: React.DragEvent) => {
      if (!options.enabled || !dragState.draggedItem) return;
      if (dragState.draggedItem.id === item.id) return;

      event.preventDefault();
      event.dataTransfer.dropEffect = options.dropEffect || 'move';

      // Calculate drop position based on mouse position
      const rect = (event.currentTarget as HTMLElement).getBoundingClientRect();
      const y = event.clientY - rect.top;
      const height = rect.height;
      
      let position: 'before' | 'after' | 'inside' = 'after';
      
      if (options.allowNesting) {
        if (y < height * 0.25) position = 'before';
        else if (y > height * 0.75) position = 'after';
        else position = 'inside';
      } else {
        position = y < height / 2 ? 'before' : 'after';
      }

      setDragState(prev => ({
        ...prev,
        dragOverItem: item,
        dropPosition: position
      }));
    };

    const handleDragLeave = (event: React.DragEvent) => {
      // Only clear if leaving the element completely
      const rect = (event.currentTarget as HTMLElement).getBoundingClientRect();
      const x = event.clientX;
      const y = event.clientY;
      
      if (x < rect.left || x > rect.right || y < rect.top || y > rect.bottom) {
        setDragState(prev => ({
          ...prev,
          dragOverItem: null,
          dropPosition: null
        }));
      }
    };

    const handleDrop = (event: React.DragEvent) => {
      if (!options.enabled || !dragState.draggedItem || !dragState.dropPosition) return;
      
      event.preventDefault();
      
      if (dragState.draggedItem.id !== item.id) {
        options.onDrop(dragState.draggedItem, item, dragState.dropPosition);
        
        // Handle reordering if callback provided
        if (options.onReorder && dragState.dropPosition !== 'inside') {
          const newItems = [...items];
          const draggedIndex = newItems.findIndex(i => i.id === dragState.draggedItem!.id);
          const targetIndex = newItems.findIndex(i => i.id === item.id);
          
          if (draggedIndex !== -1 && targetIndex !== -1) {
            const [draggedItem] = newItems.splice(draggedIndex, 1);
            const insertIndex = dragState.dropPosition === 'before' ? targetIndex : targetIndex + 1;
            newItems.splice(insertIndex, 0, draggedItem);
            options.onReorder(newItems);
          }
        }
      }
    };

    return {
      onDragOver: handleDragOver,
      onDragLeave: handleDragLeave,
      onDrop: handleDrop,
      'data-drop-id': item.id
    };
  }, [options.enabled, options.dropEffect, options.allowNesting, options.onDrop, options.onReorder, dragState.draggedItem, dragState.dropPosition, items]);

  const getDropIndicatorProps = useCallback((item: T) => {
    const isActive = dragState.dragOverItem?.id === item.id && dragState.dropPosition;
    const isDraggedOver = dragState.dragOverItem?.id === item.id;
    
    return {
      'data-drag-active': isActive,
      'data-drag-position': dragState.dropPosition,
      'data-dragged-over': isDraggedOver,
      className: `
        ${isActive ? 'drag-drop-active' : ''}
        ${isDraggedOver ? `drag-drop-${dragState.dropPosition}` : ''}
        ${dragState.draggedItem?.id === item.id ? 'drag-drop-dragging' : ''}
      `.trim()
    };
  }, [dragState.dragOverItem, dragState.dropPosition, dragState.draggedItem]);

  return {
    dragState,
    getDragHandlers,
    getDropHandlers,
    getDropIndicatorProps,
    dragPreviewRef
  };
};

// Hook for sortable lists
export const useSortable = <T extends { id: string }>(
  items: T[],
  onReorder: (items: T[]) => void,
  enabled = true
) => {
  return useDragAndDrop(items, {
    enabled,
    onDrop: () => {}, // Handled by onReorder
    onReorder,
    allowNesting: false,
    dropEffect: 'move'
  });
};

// Hook for nested drag and drop (like file trees)
export const useNestedDragAndDrop = <T extends { id: string; children?: T[] }>(
  items: T[],
  onDrop: (draggedItem: T, targetItem: T, position: 'before' | 'after' | 'inside') => void,
  enabled = true
) => {
  return useDragAndDrop(items, {
    enabled,
    onDrop,
    allowNesting: true,
    dropEffect: 'move'
  });
};