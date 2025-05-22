# sample API Reference

## Table of Contents

- [.](#)
  - [formatUser](#formatuser)
  - [add](#add)
  - [capitalize](#capitalize)
  - [constructor](#constructor)
  - [setProperty](#setproperty)
  - [getProperty](#getproperty)

## .

### formatUser

*Exported*

Formats a user object into a display string

```typescript
formatUser(user: {
  name: string;
  age?: number;
  roles: string[];
}): string
```

**Parameters:**

- `user` - *{
  name: string;
  age?: number;
  roles: string[];
}*

**Returns:** *string*

*Defined in [.:18]*

### add

Adds two numbers together and returns the result.
- **a:** First number to add
- **b:** Second number to add
**Returns:** Sum of the two numbers

```typescript
add(a: number, b: number): number
```

**Parameters:**

- `a` - *number*
- `b` - *number*

**Returns:** *number*

*Defined in [.:11]*

### capitalize

A private helper function that is not exported

```typescript
capitalize(str: string): string
```

**Parameters:**

- `str` - *string*

**Returns:** *string*

*Defined in [.:29]*

### constructor

Creates a new UserProfile instance
- **username:** The username for this profile
- **email:** Optional email address

```typescript
constructor(public username: string, private email?: string)
```

**Parameters:**

- `public username` - *string*
- `private email?` - *string*

*Defined in [.:45]*

### setProperty

Sets a property on the user profile
- **key:** Property name
- **value:** Property value

```typescript
setProperty(key: string, value: any): void
```

**Parameters:**

- `key` - *string*
- `value` - *any*

**Returns:** *void*

*Defined in [.:52]*

### getProperty

Gets a property from the user profile
- **key:** Property name to retrieve
- **defaultValue:** Value to return if property doesn't exist

```typescript
getProperty(key: string, defaultValue?: T): T | undefined
```

**Parameters:**

- `key` - *string*
- `defaultValue?` - *T*

**Returns:** *T | undefined*

*Defined in [.:61]*


