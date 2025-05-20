# OneLogin Connector Setup Guide

While developing the connector, please fill out this form. This information is needed to write docs and to help other users set up the connector.

---

## Connector capabilities

1. **What resources does the connector sync?**  
   This connector syncs:  
   — Accounts  
   — Roles  
   — Groups  
   — Application assignments

2. **Can the connector provision any resources? If so, which ones?**  
   The connector can provision:  
   — Roles

---

## Connector credentials

1. **What credentials or information are needed to set up the connector?**  
   This connector requires:  
   — OneLogin domain  
   — Client ID  
   — Client Secret

   **Args**:  
   `--domain`  
   `--client-id`  
   `--client-secret`

2. **For each item in the list above:**

   - **How does a user create or look up that credential or info?**

     1. Log in to OneLogin as an **Account Owner** or **Administrator**.
     2. Navigate to **Developers > API Credentials**.
     3. Click **New Credential**.
     4. Give the credential a name (e.g., “ConductorOne”).
     5. Under **Scopes**, select **Manage all**.
     6. Click **Save**.
     7. Once saved, copy and securely store the generated **Client ID** and **Client Secret**.
     8. Your **OneLogin domain** can be found in the URL of your OneLogin instance: `https://<your-domain>.onelogin.com`.

   - **Does the credential need any specific scopes or permissions?**  
     Yes. The credential must have the **Manage all** scope.

   - **Is the list of scopes or permissions different to sync (read) versus provision (read-write)?**  
     No. The **Manage all** scope covers both syncing and provisioning needs.

   - **What level of access or permissions does the user need in order to create the credentials?**  
     The user must have **Administrator** or **Account Owner** permissions in OneLogin.

---