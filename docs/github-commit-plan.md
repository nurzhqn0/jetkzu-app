# GitHub commit plan

Use feature ownership commits so each team member has a clear subsystem and at least 12 gRPC endpoints.

Commit order: Nurzhan, Ali, Dias, Nurassyl.

## Nurzhan

1. `feat(driver): expand driver and vehicle grpc endpoints`
   - DriverService owns 14 gRPC methods.
   - Includes vehicle update/delete, driver lists, location, status history, and rating.
2. `feat(gateway): expose driver management endpoints`
   - Adds REST mappings for driver operations.
3. `feat(web): add driver dashboard`
   - Adds driver registration, vehicle, status, location, and availability actions.

## Ali

1. `feat(user): expand auth and profile grpc endpoints`
   - UserService owns 14 gRPC methods.
   - Includes logout, validation, listing, password, verification, deactivation, and role update flows.
2. `feat(gateway): expose user endpoints and frontend cors`
   - Adds REST mappings for user/admin auth flows.
   - Adds local frontend CORS support.
3. `feat(web): add auth and profile screens`
   - Adds registration, login, profile, and token persistence UI.

## Dias

1. `feat(payment-notification): expand billing and notification endpoints`
   - PaymentService plus NotificationService own 19 gRPC methods.
   - Includes payment listing, receipts, method validation, refund request, unread notifications, read-all, resend, delete, and ride receipts.
2. `feat(compose): add frontend service`
   - Adds `web/Dockerfile`, nginx proxy, and Docker Compose frontend service.
3. `test: verify final demo flow`
   - Adds frontend API tests.
   - Updates grading checklist and demo documentation.

## Nurassyl

1. `feat(ride): expand ride lifecycle grpc endpoints`
   - RideService owns 14 gRPC methods.
   - Includes active rides, driver rides, history, scheduling, accept/reject, and rating.
2. `feat(gateway): expose ride history and lifecycle endpoints`
   - Adds REST mappings for expanded ride operations.
3. `feat(web): add passenger ride flow`
   - Adds estimate, create, active rides, complete, and history actions.
