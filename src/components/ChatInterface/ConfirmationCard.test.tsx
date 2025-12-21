/**
 * Unit tests for ConfirmationCard component
 * Requirements: 5.3, 5.4, 5.5
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, fireEvent, act } from '@testing-library/react';
import { ConfirmationCard } from './ConfirmationCard';
import type { ConfirmationRequest } from '../../types/models';

describe('ConfirmationCard', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.clearAllTimers();
  });

  it('should render confirmation request with all required elements', () => {
    const request: ConfirmationRequest = {
      id: 'test-1',
      operation: 'Delete Database',
      description: 'This will permanently delete all data',
      riskLevel: 'high',
      timeout: 30,
    };

    const { getByTestId } = render(
      <ConfirmationCard
        request={request}
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />
    );

    expect(getByTestId('operation').textContent).toBe('Delete Database');
    expect(getByTestId('description').textContent).toBe('This will permanently delete all data');
    expect(getByTestId('risk-level').textContent).toContain('high');
    expect(getByTestId('confirm-button')).toBeDefined();
    expect(getByTestId('cancel-button')).toBeDefined();
  });

  it('should call onConfirm when confirm button is clicked', () => {
    const onConfirm = vi.fn();
    const request: ConfirmationRequest = {
      id: 'test-2',
      operation: 'Restart Service',
      description: 'Service will be unavailable for a few seconds',
      riskLevel: 'medium',
      timeout: 30,
    };

    const { getByTestId } = render(
      <ConfirmationCard
        request={request}
        onConfirm={onConfirm}
        onCancel={vi.fn()}
      />
    );

    fireEvent.click(getByTestId('confirm-button'));

    expect(onConfirm).toHaveBeenCalledWith('test-2');
    expect(getByTestId('status').textContent).toBe('已确认');
  });

  it('should call onCancel when cancel button is clicked', () => {
    const onCancel = vi.fn();
    const request: ConfirmationRequest = {
      id: 'test-3',
      operation: 'Update Config',
      description: 'Configuration will be updated',
      riskLevel: 'low',
      timeout: 30,
    };

    const { getByTestId } = render(
      <ConfirmationCard
        request={request}
        onConfirm={vi.fn()}
        onCancel={onCancel}
      />
    );

    fireEvent.click(getByTestId('cancel-button'));

    expect(onCancel).toHaveBeenCalledWith('test-3');
    expect(getByTestId('status').textContent).toBe('已取消');
  });

  it('should handle timeout and call onCancel', () => {
    const onCancel = vi.fn();
    const request: ConfirmationRequest = {
      id: 'test-4',
      operation: 'Deploy Application',
      description: 'New version will be deployed',
      riskLevel: 'high',
      timeout: 3,
    };

    const { getByTestId } = render(
      <ConfirmationCard
        request={request}
        onConfirm={vi.fn()}
        onCancel={onCancel}
      />
    );

    // Initial timer value
    expect(getByTestId('timer').textContent).toBe('3秒');

    // Advance time by 1 second
    act(() => {
      vi.advanceTimersByTime(1000);
    });
    expect(getByTestId('timer').textContent).toBe('2秒');

    // Advance time by 2 more seconds (total 3 seconds)
    act(() => {
      vi.advanceTimersByTime(2000);
    });
    expect(onCancel).toHaveBeenCalledWith('test-4');
    expect(getByTestId('status').textContent).toBe('已超时');
  });

  it('should not allow confirm after cancel', () => {
    const onConfirm = vi.fn();
    const onCancel = vi.fn();
    const request: ConfirmationRequest = {
      id: 'test-5',
      operation: 'Test Operation',
      description: 'Test description',
      riskLevel: 'low',
      timeout: 30,
    };

    const { getByTestId } = render(
      <ConfirmationCard
        request={request}
        onConfirm={onConfirm}
        onCancel={onCancel}
      />
    );

    // Click cancel first
    fireEvent.click(getByTestId('cancel-button'));
    expect(onCancel).toHaveBeenCalledTimes(1);

    // Try to click confirm (button should not exist)
    const confirmButton = getByTestId('confirmation-card').querySelector('[data-testid="confirm-button"]');
    expect(confirmButton).toBeNull();
    expect(onConfirm).not.toHaveBeenCalled();
  });

  it('should not allow cancel after confirm', () => {
    const onConfirm = vi.fn();
    const onCancel = vi.fn();
    const request: ConfirmationRequest = {
      id: 'test-6',
      operation: 'Test Operation',
      description: 'Test description',
      riskLevel: 'low',
      timeout: 30,
    };

    const { getByTestId } = render(
      <ConfirmationCard
        request={request}
        onConfirm={onConfirm}
        onCancel={onCancel}
      />
    );

    // Click confirm first
    fireEvent.click(getByTestId('confirm-button'));
    expect(onConfirm).toHaveBeenCalledTimes(1);

    // Try to click cancel (button should not exist)
    const cancelButton = getByTestId('confirmation-card').querySelector('[data-testid="cancel-button"]');
    expect(cancelButton).toBeNull();
    expect(onCancel).not.toHaveBeenCalled();
  });

  it('should apply correct CSS class for high risk level', () => {
    const request: ConfirmationRequest = {
      id: 'test-7',
      operation: 'High Risk Operation',
      description: 'Very dangerous',
      riskLevel: 'high',
      timeout: 30,
    };

    const { getByTestId } = render(
      <ConfirmationCard
        request={request}
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />
    );

    const card = getByTestId('confirmation-card');
    expect(card.className).toContain('risk-high');
  });

  it('should apply correct CSS class for medium risk level', () => {
    const request: ConfirmationRequest = {
      id: 'test-8',
      operation: 'Medium Risk Operation',
      description: 'Moderately dangerous',
      riskLevel: 'medium',
      timeout: 30,
    };

    const { getByTestId } = render(
      <ConfirmationCard
        request={request}
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />
    );

    const card = getByTestId('confirmation-card');
    expect(card.className).toContain('risk-medium');
  });

  it('should apply correct CSS class for low risk level', () => {
    const request: ConfirmationRequest = {
      id: 'test-9',
      operation: 'Low Risk Operation',
      description: 'Safe operation',
      riskLevel: 'low',
      timeout: 30,
    };

    const { getByTestId } = render(
      <ConfirmationCard
        request={request}
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />
    );

    const card = getByTestId('confirmation-card');
    expect(card.className).toContain('risk-low');
  });

  it('should stop timer after confirm', () => {
    const onCancel = vi.fn();
    const request: ConfirmationRequest = {
      id: 'test-10',
      operation: 'Test Operation',
      description: 'Test description',
      riskLevel: 'low',
      timeout: 5,
    };

    const { getByTestId } = render(
      <ConfirmationCard
        request={request}
        onConfirm={vi.fn()}
        onCancel={onCancel}
      />
    );

    // Click confirm
    fireEvent.click(getByTestId('confirm-button'));

    // Advance time past timeout
    vi.advanceTimersByTime(10000);

    // onCancel should not be called because timer was stopped
    expect(onCancel).not.toHaveBeenCalled();
  });
});
