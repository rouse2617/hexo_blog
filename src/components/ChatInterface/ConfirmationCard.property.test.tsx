/**
 * Property-based tests for ConfirmationCard component
 * Requirements: 5.1, 5.2, 5.3, 5.4
 * 
 * Each property test runs 100 iterations with randomly generated inputs
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, fireEvent } from '@testing-library/react';
import { ConfirmationCard } from './ConfirmationCard';
import type { ConfirmationRequest } from '../../types/models';
import fc from 'fast-check';

describe('ConfirmationCard Property Tests', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.clearAllTimers();
  });

  // Feature: ops-genius-frontend, Property 18: 确认卡片渲染完整性
  // Validates: Requirements 5.1, 5.2
  it('Property 18: confirmation card should render with operation, description, and buttons', () => {
    fc.assert(
      fc.property(
        fc.record({
          id: fc.string().filter(s => s.length > 0),
          operation: fc.string().filter(s => s.trim().length > 0),
          description: fc.string().filter(s => s.trim().length > 0),
          riskLevel: fc.constantFrom('low', 'medium', 'high') as fc.Arbitrary<'low' | 'medium' | 'high'>,
          timeout: fc.integer({ min: 1, max: 60 }),
        }),
        (request: ConfirmationRequest) => {
          const { getByTestId, container, unmount } = render(
            <ConfirmationCard
              request={request}
              onConfirm={vi.fn()}
              onCancel={vi.fn()}
            />
          );

          // Verify operation is rendered
          const operation = getByTestId('operation');
          expect(operation.textContent).toBe(request.operation);

          // Verify description is rendered
          const description = getByTestId('description');
          expect(description.textContent).toBe(request.description);

          // Verify risk level is rendered
          const riskLevel = getByTestId('risk-level');
          expect(riskLevel.textContent).toContain(request.riskLevel);

          // Verify confirm button exists
          const confirmButton = container.querySelector('[data-testid="confirm-button"]');
          expect(confirmButton).not.toBeNull();

          // Verify cancel button exists
          const cancelButton = container.querySelector('[data-testid="cancel-button"]');
          expect(cancelButton).not.toBeNull();

          unmount();
        }
      ),
      { numRuns: 100 }
    );
  });

  // Feature: ops-genius-frontend, Property 19: 确认操作响应正确性
  // Validates: Requirements 5.3, 5.4
  it('Property 19: clicking confirm should call onConfirm with correct ID and update status', () => {
    fc.assert(
      fc.property(
        fc.record({
          id: fc.string().filter(s => s.length > 0),
          operation: fc.string().filter(s => s.trim().length > 0),
          description: fc.string().filter(s => s.trim().length > 0),
          riskLevel: fc.constantFrom('low', 'medium', 'high') as fc.Arbitrary<'low' | 'medium' | 'high'>,
          timeout: fc.integer({ min: 1, max: 60 }),
        }),
        (request: ConfirmationRequest) => {
          const onConfirm = vi.fn();
          const { getByTestId, unmount } = render(
            <ConfirmationCard
              request={request}
              onConfirm={onConfirm}
              onCancel={vi.fn()}
            />
          );

          // Click confirm button
          const confirmButton = getByTestId('confirm-button');
          fireEvent.click(confirmButton);

          // Verify onConfirm was called with correct ID
          expect(onConfirm).toHaveBeenCalledWith(request.id);
          expect(onConfirm).toHaveBeenCalledTimes(1);

          // Verify status is updated to "已确认"
          const status = getByTestId('status');
          expect(status.textContent).toBe('已确认');

          unmount();
        }
      ),
      { numRuns: 100 }
    );
  });

  // Feature: ops-genius-frontend, Property 19: 确认操作响应正确性 (Cancel)
  // Validates: Requirements 5.3, 5.4
  it('Property 19: clicking cancel should call onCancel with correct ID and update status', () => {
    fc.assert(
      fc.property(
        fc.record({
          id: fc.string().filter(s => s.length > 0),
          operation: fc.string().filter(s => s.trim().length > 0),
          description: fc.string().filter(s => s.trim().length > 0),
          riskLevel: fc.constantFrom('low', 'medium', 'high') as fc.Arbitrary<'low' | 'medium' | 'high'>,
          timeout: fc.integer({ min: 1, max: 60 }),
        }),
        (request: ConfirmationRequest) => {
          const onCancel = vi.fn();
          const { getByTestId, unmount } = render(
            <ConfirmationCard
              request={request}
              onConfirm={vi.fn()}
              onCancel={onCancel}
            />
          );

          // Click cancel button
          const cancelButton = getByTestId('cancel-button');
          fireEvent.click(cancelButton);

          // Verify onCancel was called with correct ID
          expect(onCancel).toHaveBeenCalledWith(request.id);
          expect(onCancel).toHaveBeenCalledTimes(1);

          // Verify status is updated to "已取消"
          const status = getByTestId('status');
          expect(status.textContent).toBe('已取消');

          unmount();
        }
      ),
      { numRuns: 100 }
    );
  });

  // Feature: ops-genius-frontend, Property 18: 确认卡片渲染完整性 (Risk Level CSS)
  // Validates: Requirements 5.1, 5.2
  it('Property 18: risk level should be reflected in CSS class', () => {
    fc.assert(
      fc.property(
        fc.record({
          id: fc.string().filter(s => s.length > 0),
          operation: fc.string().filter(s => s.trim().length > 0),
          description: fc.string().filter(s => s.trim().length > 0),
          riskLevel: fc.constantFrom('low', 'medium', 'high') as fc.Arbitrary<'low' | 'medium' | 'high'>,
          timeout: fc.integer({ min: 1, max: 60 }),
        }),
        (request: ConfirmationRequest) => {
          const { getByTestId, unmount } = render(
            <ConfirmationCard
              request={request}
              onConfirm={vi.fn()}
              onCancel={vi.fn()}
            />
          );

          const card = getByTestId('confirmation-card');
          const expectedClass = `risk-${request.riskLevel}`;
          expect(card.className).toContain(expectedClass);

          unmount();
        }
      ),
      { numRuns: 100 }
    );
  });

  // Feature: ops-genius-frontend, Property 19: 确认操作响应正确性 (Idempotency)
  // Validates: Requirements 5.3, 5.4
  it('Property 19: confirm/cancel should be idempotent (only trigger once)', () => {
    fc.assert(
      fc.property(
        fc.record({
          id: fc.string().filter(s => s.length > 0),
          operation: fc.string().filter(s => s.trim().length > 0),
          description: fc.string().filter(s => s.trim().length > 0),
          riskLevel: fc.constantFrom('low', 'medium', 'high') as fc.Arbitrary<'low' | 'medium' | 'high'>,
          timeout: fc.integer({ min: 1, max: 60 }),
        }),
        fc.boolean(),
        (request: ConfirmationRequest, shouldConfirm: boolean) => {
          const onConfirm = vi.fn();
          const onCancel = vi.fn();
          const { getByTestId, container, unmount } = render(
            <ConfirmationCard
              request={request}
              onConfirm={onConfirm}
              onCancel={onCancel}
            />
          );

          if (shouldConfirm) {
            // Click confirm
            const confirmButton = getByTestId('confirm-button');
            fireEvent.click(confirmButton);
            expect(onConfirm).toHaveBeenCalledTimes(1);

            // Try to click again (button should not exist)
            const confirmButtonAfter = container.querySelector('[data-testid="confirm-button"]');
            expect(confirmButtonAfter).toBeNull();

            // Cancel should not be callable
            const cancelButtonAfter = container.querySelector('[data-testid="cancel-button"]');
            expect(cancelButtonAfter).toBeNull();
            expect(onCancel).not.toHaveBeenCalled();
          } else {
            // Click cancel
            const cancelButton = getByTestId('cancel-button');
            fireEvent.click(cancelButton);
            expect(onCancel).toHaveBeenCalledTimes(1);

            // Try to click again (button should not exist)
            const cancelButtonAfter = container.querySelector('[data-testid="cancel-button"]');
            expect(cancelButtonAfter).toBeNull();

            // Confirm should not be callable
            const confirmButtonAfter = container.querySelector('[data-testid="confirm-button"]');
            expect(confirmButtonAfter).toBeNull();
            expect(onConfirm).not.toHaveBeenCalled();
          }

          unmount();
        }
      ),
      { numRuns: 100 }
    );
  });

  // Feature: ops-genius-frontend, Property 18: 确认卡片渲染完整性 (Timer Display)
  // Validates: Requirements 5.5
  it('Property 18: timer should display correct initial value', () => {
    fc.assert(
      fc.property(
        fc.record({
          id: fc.string().filter(s => s.length > 0),
          operation: fc.string().filter(s => s.trim().length > 0),
          description: fc.string().filter(s => s.trim().length > 0),
          riskLevel: fc.constantFrom('low', 'medium', 'high') as fc.Arbitrary<'low' | 'medium' | 'high'>,
          timeout: fc.integer({ min: 1, max: 60 }),
        }),
        (request: ConfirmationRequest) => {
          const { getByTestId, unmount } = render(
            <ConfirmationCard
              request={request}
              onConfirm={vi.fn()}
              onCancel={vi.fn()}
            />
          );

          const timer = getByTestId('timer');
          expect(timer.textContent).toBe(`${request.timeout}秒`);
          
          unmount();
        }
      ),
      { numRuns: 100 }
    );
  });
});
