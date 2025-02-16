## Docker Registry Authentication Fix
Documented proper PAT setup for GitHub Container Registry authentication and enabled required permissions.

- Uncommented packages write permission in release workflow
- Added documentation for setting up RELEASE_ACTION_PAT with proper scopes 

# AWS Config Debug Logging

Added debug logging for AWS configuration to aid in troubleshooting AWS connectivity issues. The logging includes non-sensitive information like region and retry mode settings.

- Added debug logging after AWS config loading in SSM evaluator 

# Enhanced AWS Debug Information

Added additional AWS debugging information to help with authentication and identity issues:

- Added truncated access key ID logging (first 4 chars only)
- Added credential provider source logging
- Added STS caller identity information (account, ARN, user ID) 

# Enhanced SSM Error Logging

Added AWS identity information to SSM parameter retrieval error logs to help diagnose permission issues:

- Added account and ARN information when SSM parameter retrieval fails
- Integrated STS identity checks into error context 

# Fix SSM Region Resolution

Fixed SSM parameter access by explicitly setting AWS region:

- Added explicit us-east-1 region configuration to match SSM parameter location
- Addresses ResolveEndpointV2 errors when region is not set in environment 

# Improve Region Configuration

Enhanced AWS region configuration to be more flexible:

- Added support for AWS_REGION and AWS_DEFAULT_REGION environment variables
- Added debug logging for region environment variables
- Fallback to us-east-1 only if no region is set in environment 