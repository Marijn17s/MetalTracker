import {ReactNode, useEffect} from 'react';
import {createPortal} from 'react-dom';

interface ModalProps {
  children: ReactNode;
  onBackdropClick?: () => void;
  className?: string;
  panelClassName?: string;
}

export function Modal({
  children,
  onBackdropClick,
  className = '',
  panelClassName = 'modal-panel',
}: ModalProps) {
  useEffect(() => {
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    return () => {
      document.body.style.overflow = previousOverflow;
    };
  }, []);

  useEffect(() => {
    function onKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape') {
        onBackdropClick?.();
      }
    }
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [onBackdropClick]);

  return createPortal(
    <div
      className={`modal-backdrop ${className}`.trim()}
      onClick={onBackdropClick}
      role="presentation"
    >
      <div
        className={`glass panel ${panelClassName}`.trim()}
        role="dialog"
        aria-modal="true"
        onClick={(event) => event.stopPropagation()}
      >
        {children}
      </div>
    </div>,
    document.body,
  );
}
