# 🔑 Keymaster

A lightweight, agentless SSH key manager that just does the job.

## What is Keymaster?

Keymaster centralizes control of your `authorized_keys` files. Fed up with
complex configuration management tools or manually scattering keys across your
fleet? Keymaster is for you. It uses a simple SQLite database as the source of
truth and a single "system key" per managed account to rewrite and
version-control access. No agents to install on remote hosts, no complex server
setup.

## Core Features

- **Modern Interactive TUI:** A beautiful and responsive terminal UI built with
    `lipgloss` that makes key management intuitive and efficient.
- **Agentless Deployment:** Uses standard SSH/SFTP to connect to hosts and manage
    `authorized_keys` files. No remote agents required.
- **Automatic System Key Hardening:** Enforces the principle of least privilege by
    automatically applying strict, SFTP-only restrictions to its own system key
    on every deployment. This is a critical, zero-config security feature.
- **Database Portability:** Easily `backup` your entire database to a compressed
    JSON file, `restore` it for disaster recovery, or `migrate` seamlessly from
    SQLite to PostgreSQL/MySQL.
- **Robust Operations:**
  - **Safe Key Rotation:** Rotate system keys without losing access to hosts
      that were offline during the change.
  - **Fleet-Wide Actions:** Deploy key updates or `audit` your entire fleet for
      configuration drift with a single command.
  - **Resilient Bootstrapping:** A crash-proof bootstrap process ensures no
      orphaned temporary keys are left on remote hosts.
- **Scriptable CLI:** All core features are available as command-line arguments,
    making Keymaster perfect for automation.
- **Multi-Language Support:** The TUI is fully internationalized, with support
    for English, German, and even Old English.
- **Flexible Backend:** Start with the default zero-config SQLite database, and
    migrate to PostgreSQL or MySQL as your needs grow.

## A New Look

Keymaster 1.2 introduces a completely redesigned user interface to make managing your keys a pleasure.

```text
   🔑 Keymaster

An agentless SSH key manager that just does the job.
╭──────────────────────────────────────╮  ╭────────────────────────────────────────────────────────────────────────────╮
│                                      │  │                                                                            │
│  Navigation                          │  │  System Status                                                             │
│                                      │  │                                                                            │
│  ▸ Manage Accounts                   │  │  Managed Accounts: 22 (22 active)                                          │
│    Manage Public Keys                │  │       Public Keys: 8 (4 global)                                            │
│    Assign Keys to Accounts           │  │        System Key: Active (Serial #3)                                      │
│    Rotate System Keys                │  │                                                                            │
│    Deploy to Fleet                   │  │                                                                            │
│    View Audit Log                    │  │  Deployment Status                                                         │
│    Audit Hosts                       │  │                                                                            │
│    View Accounts by Tag              │  │  Hosts using current key: 21                                               │
│    Language                          │  │  Hosts using past key(s): 1                                                │
│                                      │  │                                                                            │
│                                      │  │                                                                            │
│                                      │  │  Security Posture                                                          │
│                                      │  │                                                                            │
│                                      │  │  Key-Type Spread: ecdsa-sha2-nistp256: 2, ssh-ed25519: 4, ssh-rsa: 2       │
│                                      │  │                                                                            │
│                                      │  │                                                                            │
│                                      │  │  Recent Activity                                                           │
│                                      │  │                                                                            │
│                                      │  │  09-30T17:35 ROTATE_SYSTEM_KEY new_serial: 3                               │
│                                      │  │  09-30T00:51 TRUST_HOST hostname: 192.168.10.136                           │
│                                      │  │  09-30T00:51 ADD_ACCOUNT account: root@192.168.10.136                      │
│                                      │  │  09-30T00:49 TRUST_HOST hostname: 192.168.10.136                           │
│                                      │  │  09-30T00:49 TRUST_HOST hostname: 192.168.10.136                           │
│                                      │  │                                                                            │
╰──────────────────────────────────────╯  ╰────────────────────────────────────────────────────────────────────────────╯
 j/k up/down: navigate   enter: select   q: quit   L: language                         
```

## Getting Started

1. **Installation:**

    ```sh
    go install github.com/toeirei/keymaster/cmd/keymaster@latest
    ```

2. **Initialize the Database:**
    Simply run Keymaster for the first time. It will automatically create
    `keymaster.db` in the current directory.

    ```sh
    keymaster
    ```

3. **Generate System Key:**
    Inside the TUI, navigate to "Rotate System Keys" and follow the prompt to
    generate your initial system key. This is the key Keymaster will use to
    manage your hosts.

4. **Bootstrap Your First Host:**
    Manually add the new Keymaster public key (displayed in the previous step) to the `~/.ssh/authorized_keys` file of an account you want to manage.

5. **Add the Account in Keymaster:**
    In the TUI, go to "Manage Accounts" and add the account (e.g., `root@your-server`).

6. **Trust the Host:**
    Still in "Manage Accounts," select the new account and press `v` to verify and trust the host's public key.

You are now ready to manage this host with Keymaster!

## Usage

- **Interactive TUI (Default):**

  ```sh
  keymaster
  ```

- **Deploy to all hosts:**

  ```sh
  keymaster deploy
  ```

- **Audit the fleet for drift (full file comparison):**

  ```sh
  keymaster audit
  ```

- **Trust a new host:**

  ```sh
  keymaster trust-host user@new-host
  ```

- **Import keys from a file:**

  ```sh
  keymaster import /path/to/authorized_keys
  ```

- **Export SSH config:**

  ```sh
  keymaster export-ssh-client-config ~/.ssh/config
  ```

- **Decommission an account:**

  ```sh
  # Remove entire authorized_keys file
  keymaster decommission user@new-host

  # Remove only Keymaster-managed content, keep other keys
  keymaster decommission user@hostname --keep-file

  # Decommission all accounts with a specific tag
  keymaster decommission --tag env:staging

  # Skip remote cleanup (database only)
  keymaster decommission user@hostname --skip-remote

  # Force decommission even if remote cleanup fails
  keymaster decommission user@hostname --force
  ```

This project is licensed under the MIT License - see the LICENSE file for details.

## Third-Party Licenses

Keymaster utilizes several third-party libraries, each with its own license. All dependency licenses are permissive and compatible with the MIT License. For a detailed list of dependencies and their license texts, please see the `NOTICE.md` file.

## Philosophy

This tool was born out of frustration. Existing solutions for SSH key management often felt like using a sledgehammer to crack a nut—requiring complex configuration, server daemons, and constant management. This is especially true for smaller teams or homelabs where simplicity is paramount.

Keymaster is different. It's built on a simple premise:

> A tool should do the job without making you manage the tool itself.

It's designed for sysadmins and developers who want a straightforward, reliable way to control SSH access without the overhead. It's powerful enough for a fleet but simple enough for a home lab.

### A Note on Security & The System Key

Keymaster is designed for simplicity, and part of that design involves storing its own "system" private key in the database. This is what allows Keymaster to be truly agentless—it can connect to your hosts from any machine that has access to the database, without needing a separate `~/.ssh` directory or SSH agent setup.

Here's how it works and what it means for security:

- **What is stored?** The database stores the *private* key for Keymaster's
    system identity and the *public* keys of all your users. User private keys
    are **never** seen, stored, or handled by Keymaster.
- **What is deployed?** When you deploy, Keymaster only pushes *public* keys to
    the `authorized_keys` file on remote hosts.
- **What's the risk?** The primary security consideration is the database file
    itself. If an attacker gains read access to your `keymaster.db` (or the
    equivalent in Postgres/MySQL), they will have the private key that grants
    access to all managed accounts.

**Treat your `keymaster.db` file as you would any sensitive secret, like a private key itself.** Ensure it has strict file permissions (e.g., `0600`) and is stored in a secure location. This trade-off—storing one private key for the sake of simplicity—is central to the Keymaster model.

For details on reporting security vulnerabilities, please see our Security Policy.

### Automatic System Key Hardening

To minimize risk, Keymaster automatically applies strict restrictions to its system key upon every deployment. This prevents the key from being used for interactive shell access or other unintended purposes, even if the private key is compromised. This is not something you need to configure; Keymaster handles it for you to enforce the principle of least privilege.

When deployed, the Keymaster system key in the remote `authorized_keys` file will look like this
and include the current system key serial in a header for traceability:

```text
# Keymaster Managed Keys (Serial: 1)
command="internal-sftp",no-port-forwarding,no-x11-forwarding,no-agent-forwarding,no-pty ssh-ed25519 AAA... keymaster-system-key
```

This is not something you need to configure; Keymaster handles it for you to enforce the principle of least privilege.

**What these options do:**

- `command="internal-sftp"`: This is the most important restriction. It forces the key to only be used for SFTP sessions and prevents shell command execution. Keymaster's deployment and audit logic is designed to work with this restriction.
- `no-port-forwarding`, `no-x11-forwarding`, `no-agent-forwarding`: These disable various forms of SSH tunneling, further reducing the key's capabilities.
- `no-pty`: Prevents the allocation of a terminal, which is not needed for SFTP.
