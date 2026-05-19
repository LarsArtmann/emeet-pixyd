# Domain Language

A **Unified Language** for `.` — shared across Customer, Product Owner, Developer, and AI.
Inspired by Domain-Driven Design (DDD) Ubiquitous Language.

Every term below should mean the **same thing** to everyone who reads it.
If a word means something different to a developer than to a customer, define it here.

## Glossary

| Term         | Definition               | Context                        |
| ------------ | ------------------------ | ------------------------------ |
| .            | The project/product name | What we call this system       |
| Example Term | A placeholder definition | Replace with your actual terms |

## Entities

Objects with identity and lifecycle (e.g., User, Order, Account).

<!-- Add your entities here:
| Term | Definition | Context |
|------|-----------|---------|
| User | A person who interacts with the system | Customer-facing |
-->

## Value Objects

Immutable objects defined by attributes (e.g., Email, Money, Address).

<!-- Add your value objects here:
| Term | Definition | Context |
|------|-----------|---------|
| Email | A validated email address | Unique identifier for users |
-->

## Events

Things that happen in the domain (e.g., UserRegistered, PaymentProcessed).

<!-- Add your events here:
| Term | Definition | Context |
|------|-----------|---------|
| UserRegistered | A new user completed signup | Triggers welcome email |
-->

## Commands

Actions the system can perform (e.g., CreateUser, ProcessPayment).

<!-- Add your commands here:
| Term | Definition | Context |
|------|-----------|---------|
| CreateUser | Registers a new user account | Admin action |
-->

## Bounded Contexts

Subsystems with distinct vocabulary (e.g., Billing vs. Shipping).

<!-- Define contexts where the same word means different things:
| Context | Description |
|---------|------------|
| Billing | Handles payments and invoices |
-->

---

> **How to use this file:**
>
> - Keep terms concise — one clear sentence per definition
> - Update when new domain concepts emerge
> - Use these terms consistently in code, docs, and conversations
> - When in doubt about a word's meaning, check here first
