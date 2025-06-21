import React, { useState, useEffect } from 'react'
import { 
  ArrowRightIcon,
  ArrowUpIcon,
  CheckCircleIcon,
  ExclamationTriangleIcon,
  DocumentCheckIcon,
  ClockIcon,
  UserGroupIcon,
  ShieldCheckIcon,
  ChartBarIcon,
  PlayIcon,
  PauseIcon,
  XMarkIcon
} from '@heroicons/react/24/outline'

interface PromotionRequest {
  id: string
  source_env: string
  target_env: string
  status: 'pending' | 'approved' | 'rejected' | 'in_progress' | 'completed' | 'failed'
  created_at: string
  created_by: string
  approved_by?: string
  approved_at?: string
  completed_at?: string
  changes_summary: ChangeSummary
  validation_results?: ValidationResult
  approvals_required: number
  approvals_received: number
  comments: Comment[]
  rollback_id?: string
}

interface ChangeSummary {
  total_changes: number
  additions: number
  modifications: number
  deletions: number
  risk_level: 'low' | 'medium' | 'high'
  affected_features: string[]
  breaking_changes: boolean
}

interface ValidationResult {
  passed: boolean
  checks: ValidationCheck[]
  warnings: string[]
  errors: string[]
}

interface ValidationCheck {
  name: string
  status: 'passed' | 'failed' | 'warning'
  message: string
}

interface Comment {
  id: string
  user: string
  message: string
  timestamp: string
}

interface PromotionWorkflow {
  id: string
  name: string
  source_env: string
  target_env: string
  auto_approve: boolean
  required_approvers: string[]
  validation_required: boolean
  backup_required: boolean
  notification_channels: string[]
}

interface ConfigPromotionProps {
  environments: Array<{ id: string; name: string; type: string }>
  onClose: () => void
}

const ConfigPromotion: React.FC<ConfigPromotionProps> = ({
  environments,
  onClose
}) => {
  const [activeTab, setActiveTab] = useState<'new' | 'pending' | 'history'>('new')
  const [promotionRequests, setPromotionRequests] = useState<PromotionRequest[]>([])
  const [workflows, setWorkflows] = useState<PromotionWorkflow[]>([])
  const [selectedRequest, setSelectedRequest] = useState<PromotionRequest | null>(null)
  const [loading, setLoading] = useState(true)
  
  // New promotion form state
  const [newPromotion, setNewPromotion] = useState({
    source_env: '',
    target_env: '',
    description: '',
    schedule_for: '',
    notify_users: [] as string[]
  })

  useEffect(() => {
    fetchPromotionData()
  }, [])

  const fetchPromotionData = async () => {
    try {
      const [requestsRes, workflowsRes] = await Promise.all([
        fetch('/api/config/promotions'),
        fetch('/api/config/promotion-workflows')
      ])

      if (requestsRes.ok && workflowsRes.ok) {
        const requestsData = await requestsRes.json()
        const workflowsData = await workflowsRes.json()
        
        setPromotionRequests(requestsData.data || [])
        setWorkflows(workflowsData.data || [])
      }
    } catch (error) {
      console.error('Failed to fetch promotion data:', error)
    } finally {
      setLoading(false)
    }
  }

  const createPromotion = async () => {
    try {
      const response = await fetch('/api/config/promotions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(newPromotion)
      })

      if (response.ok) {
        await fetchPromotionData()
        setActiveTab('pending')
        resetNewPromotionForm()
      }
    } catch (error) {
      console.error('Failed to create promotion:', error)
    }
  }

  const approvePromotion = async (requestId: string) => {
    try {
      const response = await fetch(`/api/config/promotions/${requestId}/approve`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ comment: 'Approved' })
      })

      if (response.ok) {
        await fetchPromotionData()
      }
    } catch (error) {
      console.error('Failed to approve promotion:', error)
    }
  }

  const rejectPromotion = async (requestId: string, reason: string) => {
    try {
      const response = await fetch(`/api/config/promotions/${requestId}/reject`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ reason })
      })

      if (response.ok) {
        await fetchPromotionData()
      }
    } catch (error) {
      console.error('Failed to reject promotion:', error)
    }
  }

  const executePromotion = async (requestId: string) => {
    try {
      const response = await fetch(`/api/config/promotions/${requestId}/execute`, {
        method: 'POST'
      })

      if (response.ok) {
        await fetchPromotionData()
      }
    } catch (error) {
      console.error('Failed to execute promotion:', error)
    }
  }

  const rollbackPromotion = async (requestId: string) => {
    if (!window.confirm('Are you sure you want to rollback this promotion?')) {
      return
    }

    try {
      const response = await fetch(`/api/config/promotions/${requestId}/rollback`, {
        method: 'POST'
      })

      if (response.ok) {
        await fetchPromotionData()
      }
    } catch (error) {
      console.error('Failed to rollback promotion:', error)
    }
  }

  const resetNewPromotionForm = () => {
    setNewPromotion({
      source_env: '',
      target_env: '',
      description: '',
      schedule_for: '',
      notify_users: []
    })
  }

  const getRiskLevelColor = (level: string) => {
    switch (level) {
      case 'low': return 'text-green-600 bg-green-50'
      case 'medium': return 'text-yellow-600 bg-yellow-50'
      case 'high': return 'text-red-600 bg-red-50'
      default: return 'text-gray-600 bg-gray-50'
    }
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'completed': return 'text-green-600 bg-green-50'
      case 'approved': return 'text-blue-600 bg-blue-50'
      case 'pending': return 'text-yellow-600 bg-yellow-50'
      case 'rejected': return 'text-red-600 bg-red-50'
      case 'failed': return 'text-red-600 bg-red-50'
      case 'in_progress': return 'text-purple-600 bg-purple-50'
      default: return 'text-gray-600 bg-gray-50'
    }
  }

  if (loading) {
    return (
      <div className="h-full flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
          <p className="mt-4 text-gray-600">Loading promotion workflows...</p>
        </div>
      </div>
    )
  }

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <div className="bg-white border-b px-6 py-4">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-xl font-semibold text-gray-900">
              Configuration Promotion
            </h2>
            <p className="text-sm text-gray-600 mt-1">
              Safely promote configurations between environments
            </p>
          </div>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600"
          >
            <svg className="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
      </div>

      {/* Tabs */}
      <div className="bg-white border-b px-6">
        <nav className="flex space-x-8">
          {['new', 'pending', 'history'].map((tab) => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab as any)}
              className={`py-3 text-sm font-medium border-b-2 transition-colors ${
                activeTab === tab
                  ? 'border-blue-500 text-blue-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700'
              }`}
            >
              {tab.charAt(0).toUpperCase() + tab.slice(1)}
              {tab === 'pending' && promotionRequests.filter(r => r.status === 'pending' || r.status === 'approved').length > 0 && (
                <span className="ml-2 px-2 py-0.5 text-xs bg-yellow-100 text-yellow-700 rounded-full">
                  {promotionRequests.filter(r => r.status === 'pending' || r.status === 'approved').length}
                </span>
              )}
            </button>
          ))}
        </nav>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto">
        {activeTab === 'new' && (
          <NewPromotionForm
            environments={environments}
            workflows={workflows}
            newPromotion={newPromotion}
            setNewPromotion={setNewPromotion}
            onSubmit={createPromotion}
          />
        )}

        {activeTab === 'pending' && (
          <PendingPromotions
            requests={promotionRequests.filter(r => 
              ['pending', 'approved', 'in_progress'].includes(r.status)
            )}
            onApprove={approvePromotion}
            onReject={rejectPromotion}
            onExecute={executePromotion}
            onSelect={setSelectedRequest}
          />
        )}

        {activeTab === 'history' && (
          <PromotionHistory
            requests={promotionRequests.filter(r => 
              ['completed', 'failed', 'rejected'].includes(r.status)
            )}
            onRollback={rollbackPromotion}
            onSelect={setSelectedRequest}
          />
        )}
      </div>

      {/* Request Details Modal */}
      {selectedRequest && (
        <PromotionDetailsModal
          request={selectedRequest}
          onClose={() => setSelectedRequest(null)}
          onApprove={() => approvePromotion(selectedRequest.id)}
          onReject={(reason) => rejectPromotion(selectedRequest.id, reason)}
          onExecute={() => executePromotion(selectedRequest.id)}
          onRollback={() => rollbackPromotion(selectedRequest.id)}
        />
      )}
    </div>
  )
}

// Sub-components
const NewPromotionForm: React.FC<{
  environments: Array<{ id: string; name: string; type: string }>
  workflows: PromotionWorkflow[]
  newPromotion: any
  setNewPromotion: (promotion: any) => void
  onSubmit: () => void
}> = ({ environments, workflows, newPromotion, setNewPromotion, onSubmit }) => {
  const sourceEnv = environments.find(e => e.id === newPromotion.source_env)
  const targetEnv = environments.find(e => e.id === newPromotion.target_env)
  const applicableWorkflow = workflows.find(w => 
    w.source_env === newPromotion.source_env && 
    w.target_env === newPromotion.target_env
  )

  return (
    <div className="p-6 max-w-3xl mx-auto">
      <div className="bg-white rounded-lg shadow p-6">
        <h3 className="text-lg font-semibold text-gray-900 mb-6">
          Create New Promotion Request
        </h3>

        <div className="space-y-6">
          {/* Environment Selection */}
          <div className="grid grid-cols-2 gap-6">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Source Environment
              </label>
              <select
                value={newPromotion.source_env}
                onChange={(e) => setNewPromotion({ ...newPromotion, source_env: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="">Select source...</option>
                {environments.map(env => (
                  <option key={env.id} value={env.id}>
                    {env.name} ({env.type})
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Target Environment
              </label>
              <select
                value={newPromotion.target_env}
                onChange={(e) => setNewPromotion({ ...newPromotion, target_env: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                disabled={!newPromotion.source_env}
              >
                <option value="">Select target...</option>
                {environments
                  .filter(env => env.id !== newPromotion.source_env)
                  .map(env => (
                    <option key={env.id} value={env.id}>
                      {env.name} ({env.type})
                    </option>
                  ))}
              </select>
            </div>
          </div>

          {/* Workflow Info */}
          {applicableWorkflow && (
            <div className="bg-blue-50 rounded-md p-4">
              <h4 className="text-sm font-medium text-blue-900 mb-2">
                Workflow: {applicableWorkflow.name}
              </h4>
              <div className="text-sm text-blue-700 space-y-1">
                <p>• {applicableWorkflow.required_approvers.length} approvers required</p>
                <p>• Validation: {applicableWorkflow.validation_required ? 'Required' : 'Optional'}</p>
                <p>• Backup: {applicableWorkflow.backup_required ? 'Automatic' : 'Manual'}</p>
                {applicableWorkflow.auto_approve && (
                  <p>• Auto-approval enabled for low-risk changes</p>
                )}
              </div>
            </div>
          )}

          {/* Description */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Description / Reason for Promotion
            </label>
            <textarea
              value={newPromotion.description}
              onChange={(e) => setNewPromotion({ ...newPromotion, description: e.target.value })}
              rows={4}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="Describe the changes and reason for promotion..."
            />
          </div>

          {/* Schedule */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Schedule Promotion (Optional)
            </label>
            <input
              type="datetime-local"
              value={newPromotion.schedule_for}
              onChange={(e) => setNewPromotion({ ...newPromotion, schedule_for: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <p className="text-xs text-gray-500 mt-1">
              Leave empty to promote immediately after approval
            </p>
          </div>

          {/* Pre-flight Checks */}
          {sourceEnv && targetEnv && (
            <div className="bg-gray-50 rounded-md p-4">
              <h4 className="text-sm font-medium text-gray-900 mb-3">
                Pre-flight Checks
              </h4>
              <div className="space-y-2">
                <PreflightCheck
                  name="Configuration Validation"
                  status="pending"
                  message="Will validate configuration before promotion"
                />
                <PreflightCheck
                  name="Environment Compatibility"
                  status="pending"
                  message="Will check target environment compatibility"
                />
                <PreflightCheck
                  name="Backup Creation"
                  status="pending"
                  message="Will create backup of target environment"
                />
              </div>
            </div>
          )}

          {/* Actions */}
          <div className="flex justify-end space-x-3 pt-4 border-t">
            <button
              onClick={() => setNewPromotion({
                source_env: '',
                target_env: '',
                description: '',
                schedule_for: '',
                notify_users: []
              })}
              className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
            >
              Reset
            </button>
            <button
              onClick={onSubmit}
              disabled={!newPromotion.source_env || !newPromotion.target_env || !newPromotion.description}
              className="px-6 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Create Promotion Request
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

const PendingPromotions: React.FC<{
  requests: PromotionRequest[]
  onApprove: (id: string) => void
  onReject: (id: string, reason: string) => void
  onExecute: (id: string) => void
  onSelect: (request: PromotionRequest) => void
}> = ({ requests, onApprove, onReject, onExecute, onSelect }) => {
  if (requests.length === 0) {
    return (
      <div className="p-6 text-center">
        <CheckCircleIcon className="h-12 w-12 mx-auto text-gray-300 mb-4" />
        <p className="text-gray-500">No pending promotion requests</p>
      </div>
    )
  }

  return (
    <div className="p-6">
      <div className="space-y-4">
        {requests.map(request => (
          <PromotionCard
            key={request.id}
            request={request}
            onApprove={() => onApprove(request.id)}
            onReject={(reason) => onReject(request.id, reason)}
            onExecute={() => onExecute(request.id)}
            onSelect={() => onSelect(request)}
          />
        ))}
      </div>
    </div>
  )
}

const PromotionHistory: React.FC<{
  requests: PromotionRequest[]
  onRollback: (id: string) => void
  onSelect: (request: PromotionRequest) => void
}> = ({ requests, onRollback, onSelect }) => {
  if (requests.length === 0) {
    return (
      <div className="p-6 text-center">
        <ClockIcon className="h-12 w-12 mx-auto text-gray-300 mb-4" />
        <p className="text-gray-500">No promotion history</p>
      </div>
    )
  }

  return (
    <div className="p-6">
      <div className="space-y-4">
        {requests.map(request => (
          <div
            key={request.id}
            className="bg-white border rounded-lg p-4 hover:shadow-md transition-shadow cursor-pointer"
            onClick={() => onSelect(request)}
          >
            <div className="flex items-start justify-between">
              <div className="flex-1">
                <div className="flex items-center space-x-3">
                  <span className={`px-2 py-1 text-xs font-medium rounded-full ${getStatusColor(request.status)}`}>
                    {request.status}
                  </span>
                  <span className="text-sm text-gray-900">
                    {request.source_env} → {request.target_env}
                  </span>
                  <span className="text-sm text-gray-500">
                    {new Date(request.created_at).toLocaleDateString()}
                  </span>
                </div>
                <p className="text-sm text-gray-600 mt-1">
                  by {request.created_by}
                </p>
                <div className="mt-2 flex items-center space-x-4 text-xs text-gray-500">
                  <span>{request.changes_summary.total_changes} changes</span>
                  <span className={`px-2 py-0.5 rounded ${getRiskLevelColor(request.changes_summary.risk_level)}`}>
                    {request.changes_summary.risk_level} risk
                  </span>
                </div>
              </div>
              {request.status === 'completed' && !request.rollback_id && (
                <button
                  onClick={(e) => {
                    e.stopPropagation()
                    onRollback(request.id)
                  }}
                  className="px-3 py-1 text-sm text-red-600 hover:text-red-800"
                >
                  Rollback
                </button>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

const PromotionCard: React.FC<{
  request: PromotionRequest
  onApprove: () => void
  onReject: (reason: string) => void
  onExecute: () => void
  onSelect: () => void
}> = ({ request, onApprove, onReject, onExecute, onSelect }) => {
  const canExecute = request.status === 'approved' || 
    (request.approvals_received >= request.approvals_required)

  return (
    <div 
      className="bg-white border rounded-lg p-6 hover:shadow-md transition-shadow cursor-pointer"
      onClick={onSelect}
    >
      <div className="flex items-start justify-between">
        <div className="flex-1">
          <div className="flex items-center space-x-3 mb-2">
            <ArrowRightIcon className="h-5 w-5 text-gray-400" />
            <span className="text-lg font-medium text-gray-900">
              {request.source_env} → {request.target_env}
            </span>
            <span className={`px-2 py-1 text-xs font-medium rounded-full ${getStatusColor(request.status)}`}>
              {request.status}
            </span>
          </div>

          <p className="text-sm text-gray-600 mb-3">
            Created by {request.created_by} • {new Date(request.created_at).toLocaleString()}
          </p>

          {/* Change Summary */}
          <div className="bg-gray-50 rounded-md p-3 mb-3">
            <div className="flex items-center justify-between text-sm">
              <div className="space-y-1">
                <div className="flex items-center space-x-4">
                  <span className="text-gray-600">Total changes:</span>
                  <span className="font-medium">{request.changes_summary.total_changes}</span>
                </div>
                <div className="flex items-center space-x-4">
                  <span className="text-green-600">+{request.changes_summary.additions}</span>
                  <span className="text-yellow-600">~{request.changes_summary.modifications}</span>
                  <span className="text-red-600">-{request.changes_summary.deletions}</span>
                </div>
              </div>
              <div className={`px-3 py-1 rounded-md text-sm font-medium ${
                getRiskLevelColor(request.changes_summary.risk_level)
              }`}>
                {request.changes_summary.risk_level.toUpperCase()} RISK
              </div>
            </div>
          </div>

          {/* Approval Status */}
          <div className="flex items-center space-x-4 text-sm">
            <div className="flex items-center">
              <UserGroupIcon className="h-4 w-4 text-gray-400 mr-2" />
              <span className="text-gray-600">
                Approvals: {request.approvals_received}/{request.approvals_required}
              </span>
            </div>
            {request.validation_results && (
              <div className="flex items-center">
                {request.validation_results.passed ? (
                  <CheckCircleIcon className="h-4 w-4 text-green-500 mr-1" />
                ) : (
                  <ExclamationTriangleIcon className="h-4 w-4 text-red-500 mr-1" />
                )}
                <span className={request.validation_results.passed ? 'text-green-600' : 'text-red-600'}>
                  Validation {request.validation_results.passed ? 'passed' : 'failed'}
                </span>
              </div>
            )}
          </div>
        </div>

        {/* Actions */}
        <div className="ml-6 flex items-center space-x-2" onClick={(e) => e.stopPropagation()}>
          {request.status === 'pending' && (
            <>
              <button
                onClick={onApprove}
                className="px-3 py-1 text-sm font-medium text-white bg-green-600 rounded hover:bg-green-700"
              >
                Approve
              </button>
              <button
                onClick={() => {
                  const reason = prompt('Rejection reason:')
                  if (reason) onReject(reason)
                }}
                className="px-3 py-1 text-sm font-medium text-white bg-red-600 rounded hover:bg-red-700"
              >
                Reject
              </button>
            </>
          )}
          {canExecute && request.status !== 'in_progress' && request.status !== 'completed' && (
            <button
              onClick={onExecute}
              className="px-3 py-1 text-sm font-medium text-white bg-blue-600 rounded hover:bg-blue-700"
            >
              Execute
            </button>
          )}
          {request.status === 'in_progress' && (
            <div className="flex items-center text-purple-600">
              <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-purple-600 mr-2"></div>
              <span className="text-sm">Executing...</span>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

const PreflightCheck: React.FC<{
  name: string
  status: 'pending' | 'passed' | 'failed' | 'warning'
  message: string
}> = ({ name, status, message }) => {
  const getIcon = () => {
    switch (status) {
      case 'passed':
        return <CheckCircleIcon className="h-5 w-5 text-green-500" />
      case 'failed':
        return <XMarkIcon className="h-5 w-5 text-red-500" />
      case 'warning':
        return <ExclamationTriangleIcon className="h-5 w-5 text-yellow-500" />
      default:
        return <ClockIcon className="h-5 w-5 text-gray-400" />
    }
  }

  return (
    <div className="flex items-start space-x-3">
      {getIcon()}
      <div className="flex-1">
        <p className="text-sm font-medium text-gray-900">{name}</p>
        <p className="text-xs text-gray-600">{message}</p>
      </div>
    </div>
  )
}

const PromotionDetailsModal: React.FC<{
  request: PromotionRequest
  onClose: () => void
  onApprove: () => void
  onReject: (reason: string) => void
  onExecute: () => void
  onRollback: () => void
}> = ({ request, onClose, onApprove, onReject, onExecute, onRollback }) => {
  return (
    <div className="fixed inset-0 bg-gray-500 bg-opacity-75 flex items-center justify-center p-4 z-50">
      <div className="bg-white rounded-lg max-w-4xl w-full max-h-[90vh] overflow-hidden flex flex-col">
        {/* Header */}
        <div className="px-6 py-4 border-b">
          <div className="flex items-center justify-between">
            <h3 className="text-lg font-medium text-gray-900">
              Promotion Request Details
            </h3>
            <button
              onClick={onClose}
              className="text-gray-400 hover:text-gray-600"
            >
              <XMarkIcon className="h-6 w-6" />
            </button>
          </div>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-6">
          <div className="space-y-6">
            {/* Status and Basic Info */}
            <div className="bg-gray-50 rounded-lg p-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <p className="text-sm text-gray-500">Status</p>
                  <p className={`mt-1 text-sm font-medium px-3 py-1 rounded-full inline-block ${
                    getStatusColor(request.status)
                  }`}>
                    {request.status}
                  </p>
                </div>
                <div>
                  <p className="text-sm text-gray-500">Risk Level</p>
                  <p className={`mt-1 text-sm font-medium px-3 py-1 rounded-full inline-block ${
                    getRiskLevelColor(request.changes_summary.risk_level)
                  }`}>
                    {request.changes_summary.risk_level}
                  </p>
                </div>
                <div>
                  <p className="text-sm text-gray-500">Created By</p>
                  <p className="mt-1 text-sm font-medium">{request.created_by}</p>
                </div>
                <div>
                  <p className="text-sm text-gray-500">Created At</p>
                  <p className="mt-1 text-sm font-medium">
                    {new Date(request.created_at).toLocaleString()}
                  </p>
                </div>
              </div>
            </div>

            {/* Changes Summary */}
            <div>
              <h4 className="text-sm font-medium text-gray-900 mb-3">Changes Summary</h4>
              <div className="bg-white border rounded-lg p-4">
                <div className="grid grid-cols-3 gap-4 text-center">
                  <div>
                    <p className="text-2xl font-bold text-green-600">
                      +{request.changes_summary.additions}
                    </p>
                    <p className="text-sm text-gray-500">Additions</p>
                  </div>
                  <div>
                    <p className="text-2xl font-bold text-yellow-600">
                      ~{request.changes_summary.modifications}
                    </p>
                    <p className="text-sm text-gray-500">Modifications</p>
                  </div>
                  <div>
                    <p className="text-2xl font-bold text-red-600">
                      -{request.changes_summary.deletions}
                    </p>
                    <p className="text-sm text-gray-500">Deletions</p>
                  </div>
                </div>
                {request.changes_summary.breaking_changes && (
                  <div className="mt-4 p-3 bg-red-50 rounded-md">
                    <p className="text-sm text-red-700 font-medium">
                      ⚠️ This promotion contains breaking changes
                    </p>
                  </div>
                )}
              </div>
            </div>

            {/* Validation Results */}
            {request.validation_results && (
              <div>
                <h4 className="text-sm font-medium text-gray-900 mb-3">Validation Results</h4>
                <div className="space-y-2">
                  {request.validation_results.checks.map((check, index) => (
                    <div key={index} className="flex items-center justify-between p-3 bg-gray-50 rounded-md">
                      <div className="flex items-center">
                        {check.status === 'passed' ? (
                          <CheckCircleIcon className="h-5 w-5 text-green-500 mr-3" />
                        ) : check.status === 'warning' ? (
                          <ExclamationTriangleIcon className="h-5 w-5 text-yellow-500 mr-3" />
                        ) : (
                          <XMarkIcon className="h-5 w-5 text-red-500 mr-3" />
                        )}
                        <div>
                          <p className="text-sm font-medium">{check.name}</p>
                          <p className="text-xs text-gray-600">{check.message}</p>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Comments */}
            <div>
              <h4 className="text-sm font-medium text-gray-900 mb-3">Comments</h4>
              <div className="space-y-3">
                {request.comments.map((comment) => (
                  <div key={comment.id} className="bg-gray-50 rounded-md p-3">
                    <div className="flex items-center justify-between mb-1">
                      <p className="text-sm font-medium">{comment.user}</p>
                      <p className="text-xs text-gray-500">
                        {new Date(comment.timestamp).toLocaleString()}
                      </p>
                    </div>
                    <p className="text-sm text-gray-700">{comment.message}</p>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>

        {/* Actions */}
        <div className="px-6 py-4 border-t bg-gray-50">
          <div className="flex justify-end space-x-3">
            {request.status === 'pending' && (
              <>
                <button
                  onClick={() => {
                    const reason = prompt('Rejection reason:')
                    if (reason) {
                      onReject(reason)
                      onClose()
                    }
                  }}
                  className="px-4 py-2 text-sm font-medium text-white bg-red-600 rounded-md hover:bg-red-700"
                >
                  Reject
                </button>
                <button
                  onClick={() => {
                    onApprove()
                    onClose()
                  }}
                  className="px-4 py-2 text-sm font-medium text-white bg-green-600 rounded-md hover:bg-green-700"
                >
                  Approve
                </button>
              </>
            )}
            {(request.status === 'approved' || request.approvals_received >= request.approvals_required) && 
             request.status !== 'completed' && request.status !== 'in_progress' && (
              <button
                onClick={() => {
                  onExecute()
                  onClose()
                }}
                className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700"
              >
                Execute Promotion
              </button>
            )}
            {request.status === 'completed' && !request.rollback_id && (
              <button
                onClick={() => {
                  onRollback()
                  onClose()
                }}
                className="px-4 py-2 text-sm font-medium text-white bg-red-600 rounded-md hover:bg-red-700"
              >
                Rollback
              </button>
            )}
            <button
              onClick={onClose}
              className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
            >
              Close
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

// Helper function moved outside component
function getStatusColor(status: string): string {
  switch (status) {
    case 'completed': return 'text-green-600 bg-green-50'
    case 'approved': return 'text-blue-600 bg-blue-50'
    case 'pending': return 'text-yellow-600 bg-yellow-50'
    case 'rejected': return 'text-red-600 bg-red-50'
    case 'failed': return 'text-red-600 bg-red-50'
    case 'in_progress': return 'text-purple-600 bg-purple-50'
    default: return 'text-gray-600 bg-gray-50'
  }
}

function getRiskLevelColor(level: string): string {
  switch (level) {
    case 'low': return 'text-green-600 bg-green-50'
    case 'medium': return 'text-yellow-600 bg-yellow-50'
    case 'high': return 'text-red-600 bg-red-50'
    default: return 'text-gray-600 bg-gray-50'
  }
}

export default ConfigPromotion