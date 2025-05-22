/**
 * A simple utility module that demonstrates TypeScript functionality.
 */

/**
 * Adds two numbers together and returns the result.
 * @param a First number to add
 * @param b Second number to add
 * @returns Sum of the two numbers
 */
export function add(a: number, b: number): number {
  return a + b;
}

/**
 * Formats a user object into a display string
 */
export const formatUser = (user: {
  name: string;
  age?: number;
  roles: string[];
}): string => {
  const age = user.age ? ` (${user.age})` : "";
  const roles = user.roles.join(", ");
  return `${user.name}${age} - ${roles}`;
};

// A private helper function that is not exported
function capitalize(str: string): string {
  if (!str) return str;
  return str.charAt(0).toUpperCase() + str.slice(1);
}

/**
 * User profile class that manages user data
 */
export class UserProfile {
  private data: Record<string, any> = {};

  /**
   * Creates a new UserProfile instance
   * @param username The username for this profile
   * @param email Optional email address
   */
  constructor(public username: string, private email?: string) {}

  /**
   * Sets a property on the user profile
   * @param key Property name
   * @param value Property value
   */
  setProperty(key: string, value: any): void {
    this.data[key] = value;
  }

  /**
   * Gets a property from the user profile
   * @param key Property name to retrieve
   * @param defaultValue Value to return if property doesn't exist
   */
  getProperty<T>(key: string, defaultValue?: T): T | undefined {
    return this.data[key] ?? defaultValue;
  }
}
