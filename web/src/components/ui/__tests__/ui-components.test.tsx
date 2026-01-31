import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import * as React from 'react';
import { Button } from '../button';
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from '../card';
import { Input } from '../input';
import { Label } from '../label';
import {
  Dialog,
  DialogTrigger,
  DialogContent,
  DialogHeader,
  DialogFooter,
  DialogTitle,
  DialogDescription,
} from '../dialog';

describe('Button', () => {
  // Rendering tests
  it('should render with default variant', () => {
    render(<Button>Click me</Button>);
    expect(screen.getByText('Click me')).toBeInTheDocument();
  });

  it('should render children correctly', () => {
    render(<Button>Test Button</Button>);
    expect(screen.getByRole('button')).toHaveTextContent('Test Button');
  });

  // Variant tests
  it('should render with default variant styling', () => {
    render(<Button>Default</Button>);
    const button = screen.getByRole('button');
    expect(button).toHaveAttribute('data-slot', 'button');
  });

  it('should render with destructive variant', () => {
    render(<Button variant="destructive">Delete</Button>);
    expect(screen.getByRole('button')).toBeInTheDocument();
  });

  it('should render with outline variant', () => {
    render(<Button variant="outline">Outline</Button>);
    expect(screen.getByRole('button')).toBeInTheDocument();
  });

  it('should render with secondary variant', () => {
    render(<Button variant="secondary">Secondary</Button>);
    expect(screen.getByRole('button')).toBeInTheDocument();
  });

  it('should render with ghost variant', () => {
    render(<Button variant="ghost">Ghost</Button>);
    expect(screen.getByRole('button')).toBeInTheDocument();
  });

  it('should render with link variant', () => {
    render(<Button variant="link">Link</Button>);
    expect(screen.getByRole('button')).toBeInTheDocument();
  });

  // Size tests
  it('should render with default size', () => {
    render(<Button>Default Size</Button>);
    expect(screen.getByRole('button')).toBeInTheDocument();
  });

  it('should render with sm size', () => {
    render(<Button size="sm">Small</Button>);
    expect(screen.getByRole('button')).toBeInTheDocument();
  });

  it('should render with lg size', () => {
    render(<Button size="lg">Large</Button>);
    expect(screen.getByRole('button')).toBeInTheDocument();
  });

  it('should render with icon size', () => {
    render(<Button size="icon">+</Button>);
    expect(screen.getByRole('button')).toBeInTheDocument();
  });

  it('should render with icon-sm size', () => {
    render(<Button size="icon-sm">+</Button>);
    expect(screen.getByRole('button')).toBeInTheDocument();
  });

  it('should render with icon-lg size', () => {
    render(<Button size="icon-lg">+</Button>);
    expect(screen.getByRole('button')).toBeInTheDocument();
  });

  // Interaction tests
  it('should call onClick when clicked', () => {
    const onClick = vi.fn();
    render(<Button onClick={onClick}>Click</Button>);
    fireEvent.click(screen.getByRole('button'));
    expect(onClick).toHaveBeenCalled();
  });

  it('should call onClick with event when clicked', () => {
    const onClick = vi.fn();
    render(<Button onClick={onClick}>Click</Button>);
    fireEvent.click(screen.getByRole('button'));
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  // Disabled state tests
  it('should be disabled when disabled prop is true', () => {
    render(<Button disabled>Disabled</Button>);
    expect(screen.getByRole('button')).toBeDisabled();
  });

  it('should not call onClick when disabled', () => {
    const onClick = vi.fn();
    render(<Button disabled onClick={onClick}>Disabled</Button>);
    fireEvent.click(screen.getByRole('button'));
    expect(onClick).not.toHaveBeenCalled();
  });

  // Loading state tests
  it('should show loading spinner when isLoading is true', () => {
    render(<Button isLoading>Loading</Button>);
    const button = screen.getByRole('button');
    expect(button).toBeDisabled();
    expect(button.querySelector('svg')).toBeInTheDocument();
  });

  it('should be disabled when isLoading is true', () => {
    render(<Button isLoading>Loading</Button>);
    expect(screen.getByRole('button')).toBeDisabled();
  });

  it('should render children while loading', () => {
    render(<Button isLoading>Loading Text</Button>);
    expect(screen.getByText('Loading Text')).toBeInTheDocument();
  });

  // Note: asChild prop uses Radix Slot - tested at integration level

  // Accessibility tests
  it('should support aria-label', () => {
    render(<Button aria-label="Close dialog">X</Button>);
    expect(screen.getByLabelText('Close dialog')).toBeInTheDocument();
  });

  it('should support type attribute', () => {
    render(<Button type="submit">Submit</Button>);
    expect(screen.getByRole('button')).toHaveAttribute('type', 'submit');
  });

  it('should support type button explicitly', () => {
    render(<Button type="button">Button Type</Button>);
    expect(screen.getByRole('button')).toHaveAttribute('type', 'button');
  });

  // ClassName tests
  it('should apply custom className', () => {
    render(<Button className="custom-class">Custom</Button>);
    expect(screen.getByRole('button')).toHaveClass('custom-class');
  });
});

describe('Card', () => {
  // Card rendering tests
  it('should render Card component', () => {
    render(<Card>Card Content</Card>);
    expect(screen.getByText('Card Content')).toBeInTheDocument();
  });

  it('should render Card with custom className', () => {
    render(<Card className="custom-card">Content</Card>);
    expect(screen.getByText('Content')).toHaveClass('custom-card');
  });

  // CardHeader tests
  it('should render CardHeader', () => {
    render(
      <Card>
        <CardHeader>Header Content</CardHeader>
      </Card>
    );
    expect(screen.getByText('Header Content')).toBeInTheDocument();
  });

  it('should render CardHeader with custom className', () => {
    render(<CardHeader className="custom-header">Header</CardHeader>);
    expect(screen.getByText('Header')).toHaveClass('custom-header');
  });

  // CardTitle tests
  it('should render CardTitle as h3 element', () => {
    render(<CardTitle>Card Title</CardTitle>);
    expect(screen.getByRole('heading', { level: 3 })).toHaveTextContent('Card Title');
  });

  it('should render CardTitle with custom className', () => {
    render(<CardTitle className="custom-title">Title</CardTitle>);
    expect(screen.getByRole('heading')).toHaveClass('custom-title');
  });

  // CardDescription tests
  it('should render CardDescription', () => {
    render(<CardDescription>Description text</CardDescription>);
    expect(screen.getByText('Description text')).toBeInTheDocument();
  });

  it('should render CardDescription with custom className', () => {
    render(<CardDescription className="custom-desc">Description</CardDescription>);
    expect(screen.getByText('Description')).toHaveClass('custom-desc');
  });

  // CardContent tests
  it('should render CardContent', () => {
    render(<CardContent>Content area</CardContent>);
    expect(screen.getByText('Content area')).toBeInTheDocument();
  });

  it('should render CardContent with custom className', () => {
    render(<CardContent className="custom-content">Content</CardContent>);
    expect(screen.getByText('Content')).toHaveClass('custom-content');
  });

  // CardFooter tests
  it('should render CardFooter', () => {
    render(<CardFooter>Footer content</CardFooter>);
    expect(screen.getByText('Footer content')).toBeInTheDocument();
  });

  it('should render CardFooter with custom className', () => {
    render(<CardFooter className="custom-footer">Footer</CardFooter>);
    expect(screen.getByText('Footer')).toHaveClass('custom-footer');
  });

  // Integration tests
  it('should render complete Card with all subcomponents', () => {
    render(
      <Card>
        <CardHeader>
          <CardTitle>Card Title</CardTitle>
          <CardDescription>Card Description</CardDescription>
        </CardHeader>
        <CardContent>Main content goes here</CardContent>
        <CardFooter>Footer actions</CardFooter>
      </Card>
    );

    expect(screen.getByText('Card Title')).toBeInTheDocument();
    expect(screen.getByText('Card Description')).toBeInTheDocument();
    expect(screen.getByText('Main content goes here')).toBeInTheDocument();
    expect(screen.getByText('Footer actions')).toBeInTheDocument();
  });

  it('should forward ref to Card', () => {
    const ref = React.createRef<HTMLDivElement>();
    render(<Card ref={ref}>Content</Card>);
    expect(ref.current).toBeInstanceOf(HTMLDivElement);
  });

  it('should forward ref to CardHeader', () => {
    const ref = React.createRef<HTMLDivElement>();
    render(<CardHeader ref={ref}>Header</CardHeader>);
    expect(ref.current).toBeInstanceOf(HTMLDivElement);
  });

  it('should forward ref to CardTitle', () => {
    const ref = React.createRef<HTMLHeadingElement>();
    render(<CardTitle ref={ref}>Title</CardTitle>);
    expect(ref.current).toBeInstanceOf(HTMLHeadingElement);
  });
});

describe('Input', () => {
  // Rendering tests
  it('should render Input component', () => {
    render(<Input />);
    expect(screen.getByRole('textbox')).toBeInTheDocument();
  });

  it('should render with placeholder text', () => {
    render(<Input placeholder="Enter text" />);
    expect(screen.getByPlaceholderText('Enter text')).toBeInTheDocument();
  });

  // Value change tests
  it('should update value on change', () => {
    render(<Input />);
    const input = screen.getByRole('textbox');
    fireEvent.change(input, { target: { value: 'new value' } });
    expect(input).toHaveValue('new value');
  });

  it('should call onChange handler', () => {
    const onChange = vi.fn();
    render(<Input onChange={onChange} />);
    const input = screen.getByRole('textbox');
    fireEvent.change(input, { target: { value: 'test' } });
    expect(onChange).toHaveBeenCalled();
  });

  // Disabled state tests
  it('should be disabled when disabled prop is true', () => {
    render(<Input disabled />);
    expect(screen.getByRole('textbox')).toBeDisabled();
  });

  it('should not accept input when disabled', () => {
    render(<Input disabled value="initial" onChange={() => {}} />);
    const input = screen.getByRole('textbox');
    expect(input).toBeDisabled();
    expect(input).toHaveValue('initial');
  });

  // Type tests
  it('should render with text type explicitly', () => {
    render(<Input type="text" />);
    expect(screen.getByRole('textbox')).toHaveAttribute('type', 'text');
  });

  it('should render with email type', () => {
    render(<Input type="email" />);
    expect(screen.getByRole('textbox')).toHaveAttribute('type', 'email');
  });

  it('should render with number type', () => {
    render(<Input type="number" />);
    expect(screen.getByRole('spinbutton')).toHaveAttribute('type', 'number');
  });

  // Ref forwarding tests
  it('should forward ref to input element', () => {
    const ref = React.createRef<HTMLInputElement>();
    render(<Input ref={ref} />);
    expect(ref.current).toBeInstanceOf(HTMLInputElement);
  });

  it('should allow focusing via ref', () => {
    const ref = React.createRef<HTMLInputElement>();
    render(<Input ref={ref} />);
    ref.current?.focus();
    expect(document.activeElement).toBe(ref.current);
  });

  // ClassName tests
  it('should apply custom className', () => {
    render(<Input className="custom-input" />);
    expect(screen.getByRole('textbox')).toHaveClass('custom-input');
  });

  // Accessibility tests
  it('should support aria-label', () => {
    render(<Input aria-label="Search input" />);
    expect(screen.getByLabelText('Search input')).toBeInTheDocument();
  });

  it('should support aria-required', () => {
    render(<Input aria-required="true" />);
    expect(screen.getByRole('textbox')).toHaveAttribute('aria-required', 'true');
  });

  it('should support id attribute', () => {
    render(<Input id="username" />);
    expect(screen.getByRole('textbox')).toHaveAttribute('id', 'username');
  });

  it('should support name attribute', () => {
    render(<Input name="username" />);
    expect(screen.getByRole('textbox')).toHaveAttribute('name', 'username');
  });

  // Value/defaultValue tests
  it('should render with defaultValue', () => {
    render(<Input defaultValue="default text" />);
    expect(screen.getByRole('textbox')).toHaveValue('default text');
  });

  it('should render with controlled value', () => {
    render(<Input value="controlled value" onChange={() => {}} />);
    expect(screen.getByRole('textbox')).toHaveValue('controlled value');
  });
});

describe('Label', () => {
  // Rendering tests
  it('should render Label component', () => {
    render(<Label>Label Text</Label>);
    expect(screen.getByText('Label Text')).toBeInTheDocument();
  });

  it('should render as label element', () => {
    render(<Label>Test Label</Label>);
    expect(screen.getByText('Test Label').tagName.toLowerCase()).toBe('label');
  });

  // htmlFor tests
  it('should render with htmlFor attribute', () => {
    render(<Label htmlFor="input-id">Input Label</Label>);
    expect(screen.getByText('Input Label')).toHaveAttribute('for', 'input-id');
  });

  it('should associate label with input via htmlFor', () => {
    render(
      <>
        <Label htmlFor="username">Username</Label>
        <Input id="username" />
      </>
    );
    const label = screen.getByText('Username');
    expect(label).toHaveAttribute('for', 'username');
  });

  // ClassName tests
  it('should apply custom className', () => {
    render(<Label className="custom-label">Custom</Label>);
    expect(screen.getByText('Custom')).toHaveClass('custom-label');
  });

  // Ref forwarding tests
  it('should forward ref to label element', () => {
    const ref = React.createRef<HTMLLabelElement>();
    render(<Label ref={ref}>Ref Label</Label>);
    expect(ref.current).toBeInstanceOf(HTMLLabelElement);
  });

  // Integration with form elements
  it('should work with Input component', () => {
    render(
      <>
        <Label htmlFor="email">Email Address</Label>
        <Input id="email" type="email" />
      </>
    );
    expect(screen.getByText('Email Address')).toBeInTheDocument();
    expect(screen.getByRole('textbox')).toHaveAttribute('id', 'email');
  });

  it('should associate label with input via htmlFor attribute', () => {
    render(
      <>
        <Label htmlFor="test-input">Clickable Label</Label>
        <Input id="test-input" />
      </>
    );
    const label = screen.getByText('Clickable Label');
    const input = screen.getByRole('textbox');
    // Verify the label has the correct htmlFor attribute
    expect(label).toHaveAttribute('for', 'test-input');
    // Verify the input has the correct id
    expect(input).toHaveAttribute('id', 'test-input');
  });
});

describe('Dialog', () => {
  // Open/Close tests
  it('should not render content when closed', () => {
    render(
      <Dialog open={false}>
        <DialogContent>
          <DialogTitle>Dialog Title</DialogTitle>
          <DialogDescription>Description</DialogDescription>
        </DialogContent>
      </Dialog>
    );
    expect(screen.queryByText('Dialog Title')).not.toBeInTheDocument();
  });

  it('should render content when open', () => {
    render(
      <Dialog open={true}>
        <DialogContent>
          <DialogTitle>Dialog Title</DialogTitle>
          <DialogDescription>Description</DialogDescription>
        </DialogContent>
      </Dialog>
    );
    expect(screen.getByText('Dialog Title')).toBeInTheDocument();
  });

  it('should render DialogTitle when open', () => {
    render(
      <Dialog open={true}>
        <DialogContent>
          <DialogTitle>Test Title</DialogTitle>
        </DialogContent>
      </Dialog>
    );
    expect(screen.getByRole('heading')).toHaveTextContent('Test Title');
  });

  it('should render DialogDescription when open', () => {
    render(
      <Dialog open={true}>
        <DialogContent>
          <DialogTitle>Title</DialogTitle>
          <DialogDescription>Test Description</DialogDescription>
        </DialogContent>
      </Dialog>
    );
    expect(screen.getByText('Test Description')).toBeInTheDocument();
  });

  // DialogTrigger tests
  it('should open dialog when trigger is clicked', () => {
    render(
      <Dialog>
        <DialogTrigger>Open Dialog</DialogTrigger>
        <DialogContent>
          <DialogTitle>Dialog Title</DialogTitle>
        </DialogContent>
      </Dialog>
    );
    fireEvent.click(screen.getByText('Open Dialog'));
    expect(screen.getByText('Dialog Title')).toBeInTheDocument();
  });

  // DialogHeader tests
  it('should render DialogHeader', () => {
    render(
      <Dialog open={true}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Header Title</DialogTitle>
          </DialogHeader>
        </DialogContent>
      </Dialog>
    );
    expect(screen.getByText('Header Title')).toBeInTheDocument();
  });

  it('should render DialogHeader with custom className', () => {
    render(
      <Dialog open={true}>
        <DialogContent>
          <DialogHeader className="custom-header">
            <DialogTitle>Title</DialogTitle>
          </DialogHeader>
        </DialogContent>
      </Dialog>
    );
    expect(screen.getByText('Title').parentElement).toHaveClass('custom-header');
  });

  // DialogFooter tests
  it('should render DialogFooter', () => {
    render(
      <Dialog open={true}>
        <DialogContent>
          <DialogFooter>Footer Content</DialogFooter>
        </DialogContent>
      </Dialog>
    );
    expect(screen.getByText('Footer Content')).toBeInTheDocument();
  });

  it('should render DialogFooter with custom className', () => {
    render(
      <Dialog open={true}>
        <DialogContent>
          <DialogFooter className="custom-footer">Footer</DialogFooter>
        </DialogContent>
      </Dialog>
    );
    expect(screen.getByText('Footer')).toHaveClass('custom-footer');
  });

  // DialogClose tests
  it('should render close button in DialogContent', () => {
    render(
      <Dialog open={true}>
        <DialogContent>
          <DialogTitle>Title</DialogTitle>
        </DialogContent>
      </Dialog>
    );
    expect(screen.getByRole('button', { name: /close/i })).toBeInTheDocument();
  });

  it('should close dialog when close button is clicked', () => {
    const onOpenChange = vi.fn();
    render(
      <Dialog open={true} onOpenChange={onOpenChange}>
        <DialogContent>
          <DialogTitle>Title</DialogTitle>
        </DialogContent>
      </Dialog>
    );
    fireEvent.click(screen.getByRole('button', { name: /close/i }));
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  // Complete dialog integration test
  it('should render complete dialog with all components', () => {
    render(
      <Dialog open={true}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Complete Dialog</DialogTitle>
            <DialogDescription>This is a complete dialog</DialogDescription>
          </DialogHeader>
          <div>Dialog body content</div>
          <DialogFooter>
            <button>Cancel</button>
            <button>Confirm</button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    );

    expect(screen.getByText('Complete Dialog')).toBeInTheDocument();
    expect(screen.getByText('This is a complete dialog')).toBeInTheDocument();
    expect(screen.getByText('Dialog body content')).toBeInTheDocument();
    expect(screen.getByText('Cancel')).toBeInTheDocument();
    expect(screen.getByText('Confirm')).toBeInTheDocument();
  });

  // Accessibility tests
  it('should have aria-labelledby pointing to title', () => {
    render(
      <Dialog open={true}>
        <DialogContent>
          <DialogTitle>Accessible Title</DialogTitle>
        </DialogContent>
      </Dialog>
    );
    const dialog = screen.getByRole('dialog');
    expect(dialog).toHaveAttribute('aria-labelledby');
  });

  it('should have aria-describedby when description is present', () => {
    render(
      <Dialog open={true}>
        <DialogContent>
          <DialogTitle>Title</DialogTitle>
          <DialogDescription>Description text</DialogDescription>
        </DialogContent>
      </Dialog>
    );
    const dialog = screen.getByRole('dialog');
    expect(dialog).toHaveAttribute('aria-describedby');
  });

  // DialogTitle styling
  it('should render DialogTitle as h2 by default', () => {
    render(
      <Dialog open={true}>
        <DialogContent>
          <DialogTitle>Title</DialogTitle>
        </DialogContent>
      </Dialog>
    );
    const title = screen.getByText('Title');
    expect(title.tagName.toLowerCase()).toBe('h2');
  });

  it('should apply custom className to DialogTitle', () => {
    render(
      <Dialog open={true}>
        <DialogContent>
          <DialogTitle className="custom-title">Styled Title</DialogTitle>
        </DialogContent>
      </Dialog>
    );
    expect(screen.getByText('Styled Title')).toHaveClass('custom-title');
  });

  // DialogDescription styling
  it('should apply custom className to DialogDescription', () => {
    render(
      <Dialog open={true}>
        <DialogContent>
          <DialogTitle>Title</DialogTitle>
          <DialogDescription className="custom-desc">Styled Description</DialogDescription>
        </DialogContent>
      </Dialog>
    );
    expect(screen.getByText('Styled Description')).toHaveClass('custom-desc');
  });
});
