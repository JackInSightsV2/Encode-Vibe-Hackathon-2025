import React, { useState, useRef, useEffect } from 'react';
import { useAuth } from '../../contexts/AuthContext';
import UserProfile from './UserProfile';

const AuthStatus: React.FC = () => {
  const { isAuthenticated, user, loading } = useAuth();
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsDropdownOpen(false);
      }
    };

    if (isDropdownOpen) {
      document.addEventListener('mousedown', handleClickOutside);
      return () => document.removeEventListener('mousedown', handleClickOutside);
    }
  }, [isDropdownOpen]);

  if (loading) {
    return (
      <div className="flex items-center space-x-2">
        <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-blue-600"></div>
        <span className="text-sm text-slate-600">Loading...</span>
      </div>
    );
  }

  if (!isAuthenticated || !user) {
    return (
      <div className="flex items-center space-x-2">
        <div className="w-2 h-2 rounded-full bg-red-500"></div>
        <span className="text-sm text-slate-600">Not Authenticated</span>
      </div>
    );
  }

  return (
    <div className="relative" ref={dropdownRef}>
      {/* User Avatar and Info */}
      <button
        onClick={() => setIsDropdownOpen(!isDropdownOpen)}
        className="flex items-center space-x-2 hover:bg-slate-100 rounded-lg px-2 py-1 transition-colors"
      >
        <UserProfile isDropdownOpen={false} />
        <svg
          className={`w-4 h-4 text-slate-400 transition-transform ${
            isDropdownOpen ? 'rotate-180' : ''
          }`}
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </button>

      {/* Dropdown */}
      {isDropdownOpen && (
        <div className="absolute right-0 mt-2 z-50">
          <UserProfile 
            isDropdownOpen={true} 
            onClose={() => setIsDropdownOpen(false)} 
          />
        </div>
      )}
    </div>
  );
};

export default AuthStatus;