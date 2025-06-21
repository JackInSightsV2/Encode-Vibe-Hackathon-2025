import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import '@testing-library/jest-dom';
import { Input } from '../Input';

describe('Input', () => {
  it('renders input element', () => {
    render(<Input />);
    expect(screen.getByRole('textbox')).toBeInTheDocument();
  });

  it('renders label when provided', () => {
    render(<Input label="Email" />);
    expect(screen.getByText('Email')).toBeInTheDocument();
    expect(screen.getByLabelText('Email')).toBeInTheDocument();
  });

  it('shows error message when error prop is provided', () => {
    render(<Input error="This field is required" />);
    expect(screen.getByText('This field is required')).toBeInTheDocument();
    expect(screen.getByRole('alert')).toBeInTheDocument();
  });

  it('shows helper text when provided', () => {
    render(<Input helper="Enter your email address" />);
    expect(screen.getByText('Enter your email address')).toBeInTheDocument();
  });

  it('prioritizes error over helper text', () => {
    render(
      <Input 
        error="This field is required" 
        helper="Enter your email address" 
      />
    );
    expect(screen.getByText('This field is required')).toBeInTheDocument();
    expect(screen.queryByText('Enter your email address')).not.toBeInTheDocument();
  });

  it('applies error styling when error is present', () => {
    render(<Input error="Error message" />);
    const input = screen.getByRole('textbox');
    expect(input).toHaveClass('border-red-300', 'bg-red-50');
  });

  it('applies filled variant styling', () => {
    render(<Input variant="filled" />);
    const input = screen.getByRole('textbox');
    expect(input).toHaveClass('bg-gray-50');
  });

  it('renders icon when provided', () => {
    const icon = <span data-testid="test-icon">📧</span>;
    render(<Input icon={icon} />);
    expect(screen.getByTestId('test-icon')).toBeInTheDocument();
  });

  it('applies correct padding when icon is present', () => {
    const icon = <span data-testid="test-icon">📧</span>;
    render(<Input icon={icon} />);
    const input = screen.getByRole('textbox');
    expect(input).toHaveClass('pl-10');
  });

  it('handles value changes', () => {
    const handleChange = jest.fn();
    render(<Input onChange={handleChange} />);
    
    const input = screen.getByRole('textbox');
    fireEvent.change(input, { target: { value: 'test@example.com' } });
    
    expect(handleChange).toHaveBeenCalled();
  });

  it('handles disabled state', () => {
    render(<Input disabled />);
    const input = screen.getByRole('textbox');
    expect(input).toBeDisabled();
  });

  it('sets aria-invalid when error is present', () => {
    render(<Input error="Error message" />);
    const input = screen.getByRole('textbox');
    expect(input).toHaveAttribute('aria-invalid', 'true');
  });

  it('links input to error message with aria-describedby', () => {
    render(<Input label="Email" error="Error message" />);
    const input = screen.getByRole('textbox');
    const errorId = input.getAttribute('aria-describedby');
    expect(errorId).toBeTruthy();
    expect(screen.getByText('Error message')).toHaveAttribute('id', errorId);
  });
});