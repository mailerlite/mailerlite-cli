# MailerLite CLI

A command-line interface and interactive TUI dashboard for the [MailerLite API](https://www.mailerlite.com/). Manage subscribers, campaigns, automations, groups, forms, e-commerce, and more — all from your terminal.

## Installation

### Homebrew

```bash
brew install --cask mailerlite/tap/mailerlite
```

### GitHub Releases

Download pre-built binaries for Linux, macOS, and Windows from the [releases page](https://github.com/mailerlite/mailerlite-cli/releases).

### Go install

```bash
go install github.com/mailerlite/mailerlite-cli@latest
```

### From source

Requires Go 1.25+.

```bash
git clone https://github.com/mailerlite/mailerlite-cli.git
cd mailerlite-cli
go build -o mailerlite .
```

Move the binary to somewhere on your `$PATH`:

```bash
sudo mv mailerlite /usr/local/bin/
```

### Nix

Run directly without installing:

```bash
nix run git+ssh://git@github.com/mailerlite/mailerlite-cli.git
```

Or install into your profile:

```bash
nix profile install git+ssh://git@github.com/mailerlite/mailerlite-cli.git
```

Or add to a `flake.nix`:

```nix
{
  inputs.mailerlite-cli.url = "git+ssh://git@github.com/mailerlite/mailerlite-cli.git";
  # then use: inputs.mailerlite-cli.packages.${system}.default
}
```

## Authentication

The CLI supports two authentication methods: **OAuth** (recommended) and **API token**.

### OAuth (recommended)

```bash
mailerlite auth login
```

Running `mailerlite auth login` opens your browser to authorize the CLI with your MailerLite account via OAuth. This is the default and recommended method — no need to manually create or paste tokens. OAuth tokens are automatically refreshed when they expire.

### API Token

You can also authenticate with an API token:

```bash
mailerlite auth login --method token
```

You'll be prompted to enter your MailerLite API token. You can generate one from your [MailerLite dashboard](https://www.mailerlite.com/) under API Tokens.

### Auth status and logout

Check auth status:

```bash
mailerlite auth status
```

Log out:

```bash
mailerlite auth logout
```

### Multiple profiles

You can manage multiple profiles:

```bash
mailerlite profile add staging
mailerlite profile add production
mailerlite profile list
mailerlite profile switch staging
```

Use a specific profile for a single command:

```bash
mailerlite subscriber list --profile production
```

### Multiple accounts

If your OAuth credentials have access to multiple accounts:

```bash
mailerlite account list
mailerlite account switch <account_id>
```

### Environment variable

You can also set the API token via environment variable:

```bash
export MAILERLITE_API_TOKEN="your_token_here"
```

## Global flags

Every command supports these flags:

| Flag | Description |
|------|-------------|
| `--json` | Output raw JSON instead of formatted tables |
| `--verbose`, `-v` | Print HTTP request and response details |
| `--profile <name>` | Use a specific auth profile |
| `--yes`, `-y` | Skip confirmation prompts |
| `--help`, `-h` | Show help for any command |

## Dashboard

Launch an interactive TUI dashboard with vim-style keybindings:

```bash
mailerlite dashboard
```

The dashboard provides a lazygit-style interface with sidebar navigation between subscribers, campaigns, automations, groups, and forms. Press `?` for help or `q` to quit.

## Commands

### Subscribers

```bash
# List subscribers
mailerlite subscriber list
mailerlite subscriber list --limit 50 --status active

# Filter by email
mailerlite subscriber list --email user@example.com

# Get subscriber count
mailerlite subscriber count

# Get subscriber details (by ID or email)
mailerlite subscriber get <id_or_email>

# Create or update a subscriber
mailerlite subscriber upsert \
  --email "user@example.com" \
  --status active \
  --groups "group1_id,group2_id" \
  --fields '{"name":"John","company":"Acme"}'

# Update a subscriber
mailerlite subscriber update <id> --status unsubscribed

# Delete a subscriber
mailerlite subscriber delete <id>

# Forget a subscriber (GDPR)
mailerlite subscriber forget <id>
```

### Groups

```bash
# List groups
mailerlite group list
mailerlite group list --limit 50 --sort name

# Create a group
mailerlite group create --name "Newsletter"

# Update a group
mailerlite group update <group_id> --name "Weekly Newsletter"

# Delete a group
mailerlite group delete <group_id>

# List subscribers in a group
mailerlite group subscribers <group_id>

# Assign / unassign a subscriber
mailerlite group assign <group_id> <subscriber_id>
mailerlite group unassign <group_id> <subscriber_id>
```

### Campaigns

```bash
# List campaigns
mailerlite campaign list
mailerlite campaign list --status sent --type regular

# Get campaign details
mailerlite campaign get <campaign_id>

# Create a campaign
mailerlite campaign create \
  --name "Welcome Campaign" \
  --type regular \
  --subject "Welcome!" \
  --from "sender@yourdomain.com" \
  --from-name "Sender Name" \
  --content "<h1>Hello</h1>" \
  --groups "group_id"

# Update a campaign
mailerlite campaign update <campaign_id> --subject "Updated Subject"

# Schedule a campaign
mailerlite campaign schedule <campaign_id> \
  --delivery scheduled \
  --date 2026-03-01 \
  --hours 10 \
  --minutes 0

# Cancel a campaign
mailerlite campaign cancel <campaign_id>

# List campaign subscriber activity
mailerlite campaign subscribers <campaign_id>

# List available campaign languages
mailerlite campaign languages

# Delete a campaign
mailerlite campaign delete <campaign_id>
```

### Automations

```bash
# List automations
mailerlite automation list
mailerlite automation list --enabled true

# Get automation details
mailerlite automation get <automation_id>

# List automation subscriber activity
mailerlite automation subscribers <automation_id>
```

### Forms

```bash
# List forms
mailerlite form list
mailerlite form list --type popup --sort name

# Get form details
mailerlite form get <form_id>

# Update a form
mailerlite form update <form_id> --name "Updated Form"

# Delete a form
mailerlite form delete <form_id>

# List form subscribers
mailerlite form subscribers <form_id>
```

### Fields

```bash
# List subscriber fields
mailerlite field list

# Create a field
mailerlite field create --name "Company" --type text

# Update a field
mailerlite field update <field_id> --name "Organization"

# Delete a field
mailerlite field delete <field_id>
```

### Segments

```bash
# List segments
mailerlite segment list

# Update a segment
mailerlite segment update <segment_id> --name "VIP Customers"

# Delete a segment
mailerlite segment delete <segment_id>

# List subscribers in a segment
mailerlite segment subscribers <segment_id>
```

### Webhooks

```bash
# List webhooks
mailerlite webhook list

# Create a webhook
mailerlite webhook create \
  --name "My Webhook" \
  --url "https://example.com/webhook" \
  --events "subscriber.created,campaign.sent" \
  --enabled

# Get webhook details
mailerlite webhook get <webhook_id>

# Update a webhook
mailerlite webhook update <webhook_id> --name "Updated Webhook"

# Delete a webhook
mailerlite webhook delete <webhook_id>
```

### Timezones

```bash
# List available timezones
mailerlite timezone list
```

### E-Commerce: Shops

```bash
# List shops
mailerlite shop list

# Get shop details
mailerlite shop get <shop_id>

# Create a shop
mailerlite shop create --name "My Store" --url "https://mystore.com"

# Update a shop
mailerlite shop update <shop_id> --name "Updated Store"

# Delete a shop
mailerlite shop delete <shop_id>

# Get shop count
mailerlite shop count
```

### E-Commerce: Products

All product commands require `--shop`.

```bash
# List products
mailerlite product list --shop <shop_id>

# Get product details
mailerlite product get <product_id> --shop <shop_id>

# Create a product
mailerlite product create --shop <shop_id> \
  --name "T-Shirt" \
  --price 29.99 \
  --url "https://mystore.com/tshirt" \
  --description "A nice t-shirt"

# Update a product
mailerlite product update <product_id> --shop <shop_id> --price 24.99

# Delete a product
mailerlite product delete <product_id> --shop <shop_id>

# Get product count
mailerlite product count --shop <shop_id>
```

### E-Commerce: Categories

All category commands require `--shop`.

```bash
# List categories
mailerlite category list --shop <shop_id>

# Get category details
mailerlite category get <category_id> --shop <shop_id>

# Create a category
mailerlite category create --shop <shop_id> --name "Apparel"

# Update a category
mailerlite category update <category_id> --shop <shop_id> --name "Clothing"

# Delete a category
mailerlite category delete <category_id> --shop <shop_id>

# Get category count
mailerlite category count --shop <shop_id>

# List products in a category
mailerlite category products <category_id> --shop <shop_id>

# Assign / unassign a product
mailerlite category assign-product <category_id> --shop <shop_id> --product <product_id>
mailerlite category unassign-product <category_id> --shop <shop_id> --product <product_id>
```

### E-Commerce: Customers

All customer commands require `--shop`.

```bash
# List customers
mailerlite customer list --shop <shop_id>

# Get customer details
mailerlite customer get <customer_id> --shop <shop_id>

# Create a customer
mailerlite customer create --shop <shop_id> \
  --email "customer@example.com" \
  --first-name "John" \
  --last-name "Doe"

# Update a customer
mailerlite customer update <customer_id> --shop <shop_id> --first-name "Jane"

# Delete a customer
mailerlite customer delete <customer_id> --shop <shop_id>

# Get customer count
mailerlite customer count --shop <shop_id>
```

### E-Commerce: Orders

All order commands require `--shop`.

```bash
# List orders
mailerlite order list --shop <shop_id>

# Get order details
mailerlite order get <order_id> --shop <shop_id>

# Create an order
mailerlite order create --shop <shop_id> \
  --customer <customer_id> \
  --status complete \
  --total 59.98 \
  --currency USD \
  --items '[{"product_id":"prod1","quantity":2,"price":29.99}]'

# Update an order
mailerlite order update <order_id> --shop <shop_id> --status complete

# Delete an order
mailerlite order delete <order_id> --shop <shop_id>

# Get order count
mailerlite order count --shop <shop_id>
```

### E-Commerce: Carts

All cart commands require `--shop`.

```bash
# List carts
mailerlite cart list --shop <shop_id>

# Get cart details
mailerlite cart get <cart_id> --shop <shop_id>

# Update a cart
mailerlite cart update <cart_id> --shop <shop_id> --currency EUR

# Get cart count
mailerlite cart count --shop <shop_id>
```

### E-Commerce: Cart Items

All cart-item commands require `--shop` and `--cart`.

```bash
# List cart items
mailerlite cart-item list --shop <shop_id> --cart <cart_id>

# Get cart item details
mailerlite cart-item get <item_id> --shop <shop_id> --cart <cart_id>

# Add an item to a cart
mailerlite cart-item create --shop <shop_id> --cart <cart_id> \
  --product <product_id> \
  --quantity 2 \
  --price 29.99

# Update a cart item
mailerlite cart-item update <item_id> --shop <shop_id> --cart <cart_id> --quantity 3

# Delete a cart item
mailerlite cart-item delete <item_id> --shop <shop_id> --cart <cart_id>

# Get cart item count
mailerlite cart-item count --shop <shop_id> --cart <cart_id>
```

### E-Commerce: Bulk Import

```bash
# Import categories from JSON file
mailerlite import categories --shop <shop_id> --file categories.json

# Import products from JSON file
mailerlite import products --shop <shop_id> --file products.json

# Import orders from JSON file
mailerlite import orders --shop <shop_id> --file orders.json
```

## Shell completion

Generate shell completions for your shell:

```bash
# Bash
source <(mailerlite completion bash)

# Zsh
mailerlite completion zsh > "${fpath[1]}/_mailerlite"

# Fish
mailerlite completion fish | source

# PowerShell
mailerlite completion powershell | Out-String | Invoke-Expression
```

## JSON output

Add `--json` to any command to get raw JSON output, useful for scripting:

```bash
# Pipe to jq
mailerlite subscriber list --json | jq '.[].email'

# Extract an ID
mailerlite group create --name "Test" --json | jq -r '.id'
```

## License

See [LICENSE](LICENSE) for details.
