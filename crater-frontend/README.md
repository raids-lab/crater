# 🌋 Crater Frontend

Crater is a Kubernetes-based GPU cluster management system providing a comprehensive solution for GPU resource orchestration.

<table>
  <tr>
    <td align="center" width="45%">
      <img src="./docs/images/jupyter.gif"><br>
      <em>Jupyter Lab</em>
    </td>
    <td align="center" width="45%">
      <img src="./docs/images/ray.gif"><br>
      <em>Ray Job</em>
    </td>
  </tr>
  <tr>
    <td align="center" width="45%">
      <img src="./docs/images/monitor.gif"><br>
      <em>Monitor</em>
    </td>
    <td align="center" width="45%">
      <img src="./docs/images/datasets.gif"><br>
      <em>Models</em>
    </td>
  </tr>
</table>

## 🛠️ Environment Setup

> [!NOTE]
> Install Node.js and Pnpm: [Official Download](https://nodejs.org/en/download)

Ensure you have Node.js and pnpm installed. We recommend using [nvm](https://github.com/nvm-sh/nvm) for Node.js version management.

Verify installations:

```bash
node -v  # Should show v22.x or higher
pnpm -v   # Should show v10.x or higher
```

## 💻 Development Guide

### Project Configuration

For VS Code users:

1. Import `.vscode/React.code-profile` via `Profiles > Import Profile`
2. Install recommended extensions

For other IDEs, manually configure:

- Prettier
- ESLint
- Tailwind CSS IntelliSense

Clone and initialize:

```bash
git clone https://github.com/raids-lab/crater-frontend.git
cd crater-web-frontend
pnpm install
```

Create `.env.development` with:

```env
VITE_API_BASE_URL="http://localhost:8098/"
VITE_USE_MSW=false  # Enable API mocking
PORT=5180           # Dev server port
```

Start development server:

```bash
make run
```

### Core Technologies 🚀

- **Language**: TypeScript
- **Framework**: React 19
- **State Management**: Jotai
- **Data Fetching**: TanStack Query v5
- **Styling**: Tailwind CSS
- **UI Libraries**:
  - shadcn/ui (headless components)
  - Flowbite (Tailwind templates)
  - TanStack Table (headless tables)

### API Mocking 🧪

Use MSW for API simulation during development:

1. Set `VITE_USE_MSW=true` in `.env.development`
2. Add handlers in `src/mocks/handlers.ts`

### Dependency Management 📦

Check updates:

```bash
pnpm outdated
```

Update dependencies:

```bash
pnpm update       # Minor updates
pnpm update --latest  # Major updates (use cautiously)
```

Update shadcn components:

```bash
for file in src/components/ui/*.tsx; do
  pnpm dlx shadcn-ui@latest add -y $(basename "$file" .tsx)
done
```

## 🚀 Deployment

To deploy Crater Project in a production environment, we provide a Helm Chart available at: [Crater Helm Chart](https://github.com/raids-lab/crater).

Please refer to the main documentation for detailed deployment instructions.

## 📁 Project Structure

```
src/
├── components/           # Reusable components
│   ├── custom/           # Custom components
│   ├── layout/           # App layouts
│   └── ui/               # shadcn components
├── hooks/                # Custom hooks
├── lib/                  # Utilities
├── pages/                # Route-based pages
│   ├── Admin/            # Admin interfaces
│   ├── Portal/           # Job management
│   └── ...               # Other sections
├── services/             # API services
├── stores/               # State management
├── types/                # TypeScript types
└── ...
```

## 🐛 Known Issues

1. **Dark Mode Input Styling**: Browser autofill causes white backgrounds in dark mode ([TailwindCSS#8679](https://github.com/tailwindlabs/tailwindcss/discussions/8679))

## 👥 Contribution Guide

We welcome and appreciate contributions from the community! Here's how you can help improve Crater Frontend.

### 🛠️ Development Workflow

1. **Fork** the repository
2. **Clone** your fork locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/crater-frontend.git
   cd crater-frontend
   ```
3. Create a new **feature branch**:
   ```bash
   git checkout -b feat/your-feature-name
   ```
4. Make your changes and **commit** them (see commit guidelines below)
5. **Push** to your fork:
   ```bash
   git push origin feat/your-feature-name
   ```
6. Open a **Pull Request** to the main repository

### ✍️ Commit Guidelines

> [!NOTE]
> This project uses **Husky** and **Commitlint** to enforce commit message conventions.
>
> **Do not use GUI clients (GitHub Desktop, GitKraken, etc.) for committing code** as they may bypass our commit message validation hooks.

Each commit message must follow this format:

```
type(scope): subject
```

**Examples:**

```
feat(portal): add job submission form
fix(admin): resolve user role validation issue
docs(readme): update contribution guidelines
```

Allowed Types:

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style/formatting
- `refactor`: Code refactoring
- `test`: Test additions/modifications
- `chore`: Build process or tooling changes

Scope (optional):

- Indicate which part of the application is affected (e.g., `portal`, `admin`, `ui`, `api`)

## 🚨 Reporting Issues

When reporting bugs, please include:

- Steps to reproduce
- Expected vs actual behavior
- Screenshots if applicable
- Browser/OS version information

Thank you for contributing to Crater Frontend! Your help makes this project better for everyone.
