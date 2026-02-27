# @acme/core

Shared TypeScript types, constants, and utility functions for the Acme Platform.

## Usage

```typescript
import { isValidEmail, CONSTANTS } from '@acme/core';
import type { User, ApiResponse } from '@acme/core';
```

## Exports

- **Types**: `User`, `UserRole`, `Job`, `ApiResponse`, `PaginatedResult`, etc.
- **Constants**: `CONSTANTS` object with platform-wide limits
- **Utilities**: `isValidEmail()`, `clamp()`, `createError()`