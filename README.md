# 🔑 Keymaster

A lightweight, agentless SSH key manager that just does the job.

## What is Keymaster?

Keymaster centralizes control of your `authorized_keys` files. Fed up with complex configuration management tools or manually scattering keys across your fleet? Keymaster is for you. It uses a simple SQLite database as the source of truth and a single "system key" per managed account to rewrite and version-control access. No agents to install on remote hosts, no complex server setup.

## Core Features

- **Modern Interactive TUI:** A beautiful and responsive terminal UI built with `lipgloss`. Features a main dashboard, filterable lists, color-coded tables, and a consistent, professional design.
- **Centralized Management:** A single SQLite database (`keymaster.db`) acts as the source of truth for all public keys and account assignments.
- **Agentless Deployment:** Uses standard SSH/SFTP to connect to hosts and manage `authorized_keys` files. No remote agents required.
- **Safe Key Rotation:** Features a robust system key rotation mechanism. Old keys are retained to ensure you can always regain access to hosts that were offline during a rotation.
- **Fleet-Wide Operations:** Deploy key changes or audit your entire fleet of active hosts with a single command.
- **Drift Detection:** The `audit` command quickly checks all hosts to ensure their deployed keys match the central database state.
- **Scriptable CLI:** All core features are available as command-line arguments, making Keymaster perfect for automation.
- **SSH Agent Integration:** Seamlessly uses your running SSH agent (including Pageant/gpg-agent on Windows) to bootstrap new hosts without manual key copying.

## A New Look

Keymaster 1.2 introduces a completely redesigned user interface to make managing your keys a pleasure.

```
      🔑 Keymaster
  An agentless SSH key manager that just does the job.

╭──────────────────────────────────────╮  ╭──────────────────────────────────────────────────╮
│ Navigation                           │  │ System Status                                    │
│                                      │  │                                                  │
│   ▸ Manage Accounts                  │  │ Managed Accounts: 5 (5 active)                   │
│     Manage Public Keys               │  │    Public Keys: 3 (1 global)                     │
│     Assign Keys to Accounts          │  │     System Key: Active (Serial #1)               │
│     Rotate System Keys               │  │                                                  │
│     Deploy to Fleet                  │  │                                                  │
│     View Audit Log                   │  │ Recent Activity                                  │
│     View Accounts by Tag             │  │                                                  │
│                                      │  │ 09-23 10:45  ADD_ACCOUNT      account: new@host   │
│                                      │  │ 09-23 10:44  TRUST_HOST       hostname: new@host  │
│                                      │  │ 09-23 10:42  ROTATE_SYSTEM_KEY  new_serial: 1     │
│                                      │  │ 09-23 10:40  DELETE_PUBLIC_KEY  comment: old-key  │
│                                      │  │ 09-23 10:39  ADD_PUBLIC_KEY   comment: new-key    │
╰──────────────────────────────────────╯  ╰──────────────────────────────────────────────────╯

  j/k up/down: navigate  enter: select  q: quit
```

## Getting Started

1.  **Installation:**
    ```sh
    go install github.com/toeirei/keymaster/cmd/keymaster@latest
    ```

2.  **Initialize the Database:**
    Simply run Keymaster for the first time. It will automatically create `keymaster.db` in the current directory.
    ```sh
    keymaster
    ```

3.  **Generate System Key:**
    Inside the TUI, navigate to "Rotate System Keys" and follow the prompt to generate your initial system key. This is the key Keymaster will use to manage your hosts.

4.  **Bootstrap Your First Host:**
    Manually add the new Keymaster public key (displayed in the previous step) to the `~/.ssh/authorized_keys` file of an account you want to manage.

5.  **Add the Account in Keymaster:**
    In the TUI, go to "Manage Accounts" and add the account (e.g., `root@your-server`).

6.  **Trust the Host:**
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

- **Audit the fleet for drift:**
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

## The Keymaster Philosophy

This tool was born out of frustration. Existing solutions for SSH key management often felt like using a sledgehammer to crack a nut—requiring complex configuration, server daemons, and constant management.

Keymaster is different. It's built on a simple premise:

> A tool should do the job without making you manage the tool itself.

It's designed for sysadmins and developers who want a straightforward, reliable way to control SSH access without the overhead. It's powerful enough for a fleet but simple enough for a home lab.