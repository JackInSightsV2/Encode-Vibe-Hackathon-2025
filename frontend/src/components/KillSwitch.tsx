import React, { useState, useEffect } from 'react'

interface KillSwitchData {
  blocked_users: string[]
  blocked_sessions: string[]
}

const KillSwitch: React.FC = () => {
  const [data, setData] = useState<KillSwitchData>({ blocked_users: [], blocked_sessions: [] })
  const [loading, setLoading] = useState(true)
  const [newUserId, setNewUserId] = useState('')

  useEffect(() => {
    fetchKillSwitchData()
  }, [])

  const fetchKillSwitchData = async () => {
    try {
      const response = await fetch('/api/killswitch')
      if (response.ok) {
        const result = await response.json()
        setData(result.data)
      }
    } catch (error) {
      console.error('Failed to fetch kill switch data:', error)
    } finally {
      setLoading(false)
    }
  }

  const blockUser = async () => {
    if (!newUserId.trim()) return
    
    try {
      const response = await fetch('/api/killswitch', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          action: 'block',
          type: 'user',
          id: newUserId.trim()
        })
      })
      
      if (response.ok) {
        setNewUserId('')
        fetchKillSwitchData()
      }
    } catch (error) {
      console.error('Failed to block user:', error)
    }
  }

  const unblockUser = async (userId: string) => {
    try {
      const response = await fetch('/api/killswitch', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          action: 'unblock',
          type: 'user',
          id: userId
        })
      })
      
      if (response.ok) {
        fetchKillSwitchData()
      }
    } catch (error) {
      console.error('Failed to unblock user:', error)
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-8">
        <div className="text-lg">Loading kill switch data...</div>
      </div>
    )
  }

  return (
    <div>
      <header className="mb-8">
        <h1 className="text-3xl font-bold text-gray-900 mb-2">Kill Switch</h1>
        <p className="text-gray-600">Block specific users or sessions from using the middleware</p>
      </header>

      <div className="bg-white rounded-lg shadow p-6">
        <h2 className="text-xl font-bold mb-4">Block User</h2>
        
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              User ID
            </label>
            <div className="flex space-x-2">
              <input
                type="text"
                value={newUserId}
                onChange={(e) => setNewUserId(e.target.value)}
                placeholder="Enter user ID to block"
                className="flex-1 px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-red-500"
                onKeyPress={(e) => e.key === 'Enter' && blockUser()}
              />
              <button
                onClick={blockUser}
                disabled={!newUserId.trim()}
                className="bg-red-600 text-white px-4 py-2 rounded-md hover:bg-red-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                Block
              </button>
            </div>
          </div>
          
          <div>
            <h3 className="text-lg font-semibold mb-2">Currently Blocked Users</h3>
            {data.blocked_users.length === 0 ? (
              <p className="text-gray-500 text-sm">No users are currently blocked</p>
            ) : (
              <div className="space-y-2">
                {data.blocked_users.map((userId, index) => (
                  <div key={index} className="flex items-center justify-between bg-red-50 border border-red-200 rounded-md p-3">
                    <span className="font-medium text-red-800">{userId}</span>
                    <button
                      onClick={() => unblockUser(userId)}
                      className="text-red-600 hover:text-red-800 text-sm font-medium"
                    >
                      Unblock
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

export default KillSwitch