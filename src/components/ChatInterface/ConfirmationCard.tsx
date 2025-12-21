/**
 * ConfirmationCard Component
 * Requirements: 5.1, 5.2, 5.3, 5.4, 5.5
 * 
 * Displays a confirmation request for high-risk operations with:
 * - Operation description
 * - Confirm and Cancel buttons
 * - Timeout mechanism (30 seconds)
 * - Visual feedback for risk level
 */

import { useEffect, useState } from 'react';
import type { ConfirmationRequest } from '../../types/models';
import './ConfirmationCard.css';

export interface ConfirmationCardProps {
  request: ConfirmationRequest;
  onConfirm: (id: string) => void;
  onCancel: (id: string) => void;
}

export function ConfirmationCard({ request, onConfirm, onCancel }: ConfirmationCardProps) {
  const [timeLeft, setTimeLeft] = useState(request.timeout);
  const [status, setStatus] = useState<'pending' | 'confirmed' | 'cancelled' | 'timeout'>('pending');

  useEffect(() => {
    if (status !== 'pending') return;

    const interval = setInterval(() => {
      setTimeLeft((prev) => {
        if (prev <= 1) {
          clearInterval(interval);
          setStatus('timeout');
          onCancel(request.id);
          return 0;
        }
        return prev - 1;
      });
    }, 1000);

    return () => clearInterval(interval);
  }, [status, request.id, onCancel]);

  const handleConfirm = () => {
    if (status !== 'pending') return;
    setStatus('confirmed');
    onConfirm(request.id);
  };

  const handleCancel = () => {
    if (status !== 'pending') return;
    setStatus('cancelled');
    onCancel(request.id);
  };

  const getRiskLevelClass = () => {
    switch (request.riskLevel) {
      case 'high':
        return 'risk-high';
      case 'medium':
        return 'risk-medium';
      case 'low':
        return 'risk-low';
      default:
        return '';
    }
  };

  const getStatusText = () => {
    switch (status) {
      case 'confirmed':
        return '已确认';
      case 'cancelled':
        return '已取消';
      case 'timeout':
        return '已超时';
      default:
        return '';
    }
  };

  return (
    <div className={`confirmation-card ${getRiskLevelClass()}`} data-testid="confirmation-card">
      <div className="confirmation-header">
        <span className="confirmation-title">需要确认</span>
        {status === 'pending' && (
          <span className="confirmation-timer" data-testid="timer">
            {timeLeft}秒
          </span>
        )}
      </div>

      <div className="confirmation-body">
        <div className="confirmation-operation" data-testid="operation">
          {request.operation}
        </div>
        <div className="confirmation-description" data-testid="description">
          {request.description}
        </div>
        <div className="confirmation-risk" data-testid="risk-level">
          风险级别: {request.riskLevel}
        </div>
      </div>

      {status === 'pending' ? (
        <div className="confirmation-actions">
          <button
            className="btn-confirm"
            onClick={handleConfirm}
            data-testid="confirm-button"
          >
            确认
          </button>
          <button
            className="btn-cancel"
            onClick={handleCancel}
            data-testid="cancel-button"
          >
            取消
          </button>
        </div>
      ) : (
        <div className="confirmation-status" data-testid="status">
          {getStatusText()}
        </div>
      )}
    </div>
  );
}
