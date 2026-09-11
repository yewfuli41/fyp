# 2026-fl-bookit



## How to Start

1. Clone or download this project.
2. Add a <kbd>.env</kbd> file with the following:
    ```
    JWT_SECRET=replace-with-your-secret

    SENDGRID_API_KEY=replace-with-your-key
    RESEND_API_KEY=replace-with-your-key
    FROM_EMAIL=replace-with-your-email
    FROM_NAME=replace-with-your-name
    ```
    Replace `JWT_SECRET` with a random string of your choice. Email configuration is optional. If email notifications are not required, leave `SENDGRID_API_KEY`, `RESEND_API_KEY`, `FROM_EMAIL`, and `FROM_NAME` with their placeholder values. The system will continue to function normally without email configuration.
3. Open a terminal in the project root directory.
4. Start the application:
    ``` 
    docker compose up --build
    ```
5. Once the containers have started, open:
http://localhost:8080

**Note**: Test data is reset each time the application is restarted.

## How to Stop

1. Press <kbd>Ctrl</kbd> + <kbd>C</kbd> in the terminal.
2. Run:
    ```
    docker compose down
    ```

## Test Data

- **Owner account**
    - `owner@test.com`
- **Staff account** 
    - `staff1@test.com`
    - `staff2@test.com`
    - `staff3@test.com`
- **Customer account**
    - `customer1@test.com`
    - `customer2@test.com`
    - `customer3@test.com`
    - `customer4@test.com`
    - `customer5@test.com`
    - `customer6@test.com`
    
All are using the same password: **`Password123!`**.