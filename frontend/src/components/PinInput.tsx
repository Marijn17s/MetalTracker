import {ClipboardEvent, KeyboardEvent, useEffect, useRef} from 'react';
import {useLocale} from '../i18n/LocaleContext';

interface PinInputProps {
  value: string;
  onChange: (value: string) => void;
  onComplete?: (value: string) => void;
  disabled?: boolean;
  autoFocus?: boolean;
  hasError?: boolean;
  ariaLabel?: string;
  id?: string;
  /** Smaller cells for dense forms (e.g. Settings). Lock screen should omit this. */
  compact?: boolean;
}

const PIN_LENGTH = 6;
const EMPTY_SLOT = ' ';

export function isCompletePin(value: string): boolean {
  return /^\d{6}$/.test(value);
}

function digitsOnly(value: string): string {
  return value.replace(/\D/g, '').slice(0, PIN_LENGTH);
}

function toSlots(value: string): string[] {
  if (!value) {
    return Array.from({length: PIN_LENGTH}, () => '');
  }
  // Position-preserving form from this component (digits and spaces).
  if (value.length === PIN_LENGTH && /^[\d ]+$/.test(value)) {
    return value.split('').map((character) => (character === EMPTY_SLOT ? '' : character));
  }
  // Compact digits (e.g. cleared/reset from parent): left-align.
  const digits = digitsOnly(value);
  return Array.from({length: PIN_LENGTH}, (_, index) => digits[index] || '');
}

function fromSlots(slots: string[]): string {
  if (slots.every((slot) => slot === '')) {
    return '';
  }
  return slots.map((slot) => (slot === '' ? EMPTY_SLOT : slot)).join('');
}

export function PinInput({
  value,
  onChange,
  onComplete,
  disabled = false,
  autoFocus = false,
  hasError = false,
  ariaLabel,
  id = 'pin',
  compact = false,
}: PinInputProps) {
  const {t} = useLocale();
  const resolvedAriaLabel = ariaLabel || t('lock.pinLabel');
  const inputsRef = useRef<Array<HTMLInputElement | null>>([]);
  const completeFiredForRef = useRef('');

  useEffect(() => {
    if (!autoFocus || disabled) return;
    inputsRef.current[0]?.focus();
  }, [autoFocus, disabled]);

  useEffect(() => {
    if (!isCompletePin(value)) {
      completeFiredForRef.current = '';
    }
  }, [value]);

  function focusIndex(index: number) {
    const clamped = Math.max(0, Math.min(PIN_LENGTH - 1, index));
    inputsRef.current[clamped]?.focus();
    inputsRef.current[clamped]?.select();
  }

  function emitSlots(slots: string[]) {
    const encoded = fromSlots(slots);
    onChange(encoded);
    const completed = slots.every((slot) => slot !== '') ? slots.join('') : '';
    if (completed && completeFiredForRef.current !== completed) {
      completeFiredForRef.current = completed;
      onComplete?.(completed);
    }
  }

  function setDigitAt(index: number, digit: string) {
    const slots = toSlots(value);
    slots[index] = digit;
    emitSlots(slots);
  }

  function handleChange(index: number, raw: string) {
    const cleaned = digitsOnly(raw);
    if (!cleaned) {
      setDigitAt(index, '');
      return;
    }
    if (cleaned.length > 1) {
      const slots = toSlots(value);
      for (let offset = 0; offset < cleaned.length && index + offset < PIN_LENGTH; offset++) {
        slots[index + offset] = cleaned[offset];
      }
      emitSlots(slots);
      focusIndex(Math.min(PIN_LENGTH - 1, index + cleaned.length - 1));
      return;
    }
    setDigitAt(index, cleaned);
    if (index < PIN_LENGTH - 1) {
      focusIndex(index + 1);
    }
  }

  function handleKeyDown(index: number, event: KeyboardEvent<HTMLInputElement>) {
    if (event.key === 'Backspace') {
      const slots = toSlots(value);
      if (slots[index]) {
        setDigitAt(index, '');
      } else if (index > 0) {
        setDigitAt(index - 1, '');
        focusIndex(index - 1);
      }
      event.preventDefault();
      return;
    }
    if (event.key === 'ArrowLeft' && index > 0) {
      focusIndex(index - 1);
      event.preventDefault();
    }
    if (event.key === 'ArrowRight' && index < PIN_LENGTH - 1) {
      focusIndex(index + 1);
      event.preventDefault();
    }
  }

  function handlePaste(index: number, event: ClipboardEvent<HTMLInputElement>) {
    event.preventDefault();
    const pasted = digitsOnly(event.clipboardData.getData('text'));
    if (!pasted) return;
    const slots = toSlots(value);
    for (let offset = 0; offset < pasted.length && index + offset < PIN_LENGTH; offset++) {
      slots[index + offset] = pasted[offset];
    }
    emitSlots(slots);
    focusIndex(Math.min(PIN_LENGTH - 1, index + pasted.length - 1));
  }

  const slots = toSlots(value);

  return (
    <div
      className={`pin-input ${compact ? 'pin-input-compact' : ''} ${hasError ? 'pin-input-error' : ''}`}
      role="group"
      aria-label={resolvedAriaLabel}
    >
      {slots.map((digit, index) => (
        <input
          key={`${id}-${index}`}
          ref={(element) => {
            inputsRef.current[index] = element;
          }}
          id={index === 0 ? id : undefined}
          className={`pin-cell ${digit ? 'filled' : ''}`}
          type="text"
          inputMode="numeric"
          pattern="[0-9]*"
          autoComplete="off"
          autoCorrect="off"
          spellCheck={false}
          maxLength={PIN_LENGTH}
          value={digit}
          disabled={disabled}
          aria-label={`${resolvedAriaLabel} ${index + 1}`}
          onChange={(event) => handleChange(index, event.target.value)}
          onKeyDown={(event) => handleKeyDown(index, event)}
          onPaste={(event) => handlePaste(index, event)}
          onFocus={(event) => event.target.select()}
        />
      ))}
    </div>
  );
}
