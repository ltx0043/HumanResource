# HumanResources Smart Contract Documentation

## Overview

The HumanResources smart contract is a decentralized payroll management system implemented on the Optimism network. It enables organizations to manage employee registrations and process salary payments in either USDC or ETH, leveraging Chainlink price feeds and Uniswap V3 for currency conversions.

## Architecture

### Core Components

- **Contract**: `HumanResources.sol`
- **Network**: Optimism
- **Solidity Version**: ^0.8.24

### External Integrations

- **Chainlink Price Feed**: ETH/USD oracle for accurate price conversion
- **Uniswap V3**: Facilitates USDC-ETH swaps for salary payments
- **WETH**: Wrapped ETH for DEX interactions
- **USDC**: Stablecoin for salary denomination

### Key Addresses (Optimism)

```solidity
USDC: 0x0b2C639c533813f4Aa9D7837CAf62653d097Ff85
WETH: 0x4200000000000000000000000000000000000006
Chainlink ETH/USD: 0x13e3Ee699D1909E989722E753853AE30b17e08c5
Uniswap Router: 0xE592427A0AEce92De3Edee1F18E0157C05861564
```

## Interface Implementation

### Employee Management Functions

#### `registerEmployee`
```solidity
function registerEmployee(address employee, uint256 weeklyUsdSalary) external
```
- **Description**: Registers a new employee with their weekly salary in USD
- **Access**: Only HR Manager
- **Parameters**:
  - `employee`: Employee's wallet address
  - `weeklyUsdSalary`: Weekly salary amount in USD (18 decimals)
- **Events**: Emits `EmployeeRegistered`
- **Security**: Prevents duplicate registrations of active employees
- **State Changes**: Increments active employee count

#### `terminateEmployee`
```solidity
function terminateEmployee(address employee) external
```
- **Description**: Terminates an employee's contract
- **Access**: Only HR Manager
- **Parameters**:
  - `employee`: Address of employee to terminate
- **Events**: Emits `EmployeeTerminated`
- **Effects**: 
  - Decrements active employee count
  - Sets termination timestamp
- **Validation**: Checks for registered and non-terminated employee

### Salary Management Functions

#### `withdrawSalary`
```solidity
function withdrawSalary() external
```
- **Description**: Processes salary payment in preferred currency (USDC/ETH)
- **Access**: Only registered employees
- **Features**:
  - Calculates pro-rated salary based on employment duration
  - Handles currency conversion if ETH is preferred
  - Implements slippage protection (2%)
- **Security**: Uses ReentrancyGuard
- **Validation**: Checks for available salary amount
- **Events**: Emits `SalaryWithdrawn`

#### `switchCurrency`
```solidity
function switchCurrency() external
```
- **Description**: Toggles employee's preferred payment currency
- **Access**: Only registered employees
- **Process**: 
  1. Checks employee is not terminated
  2. Withdraws available salary in current currency
  3. Updates currency preference
- **Events**: 
  - Emits `SalaryWithdrawn` if salary was available
  - Emits `CurrencySwitched`
- **Security**: Uses ReentrancyGuard

### Query Functions

#### `salaryAvailable`
```solidity
function salaryAvailable(address employee) external view returns (uint256)
```
- **Description**: Calculates available salary for withdrawal
- **Returns**: Amount in USD (18 decimals)
- **Calculation**: Pro-rated based on time worked since last withdrawal

#### `hrManager`
```solidity
function hrManager() external view returns (address)
```
- **Description**: Returns HR manager's address
- **Usage**: For verification and transparency

#### `getActiveEmployeeCount`
```solidity
function getActiveEmployeeCount() external view returns (uint256)
```
- **Description**: Returns current number of active employees
- **Updates**: Automatically maintained during registration/termination

#### `getEmployeeInfo`
```solidity
function getEmployeeInfo(address employee) external view returns (uint256, uint256, uint256)
```
- **Description**: Retrieves employee details
- **Returns**: Tuple of (weeklyUsdSalary, employedSince, terminatedAt)

## AMM Integration

The contract integrates with Uniswap V3 for USDC-ETH conversions:

1. **Price Determination**:
   - Uses Chainlink ETH/USD price feed for accurate conversion rates
   - Implements 2% slippage protection

2. **Swap Process**:
   ```solidity
   ISwapRouter.ExactInputSingleParams memory params = ISwapRouter.ExactInputSingleParams({
       tokenIn: address(USDC),
       tokenOut: address(WETH),
       fee: POOL_FEE_TIER,
       recipient: self,
       deadline: block.timestamp + SWAP_DEADLINE,
       amountIn: usdcAmount,
       amountOutMinimum: minOutputAmount,
       sqrtPriceLimitX96: 0
   });
   ```

## Oracle Integration

Chainlink price feed integration for reliable ETH/USD conversion:

```solidity
(,int256 latestPrice,,,) = CHAINLINK_ETH_USDC.latestRoundData();
uint256 estimatedWeiOutput = (usdcAmount * PRICE_PRECISION) / uint256(latestPrice);
```

## Security Features

### Access Control
```solidity
modifier onlyHrManager() {
    if (msg.sender != hrManagerAddress) revert NotAuthorized();
    _;
}

modifier onlyRegisteredEmployee() {
    if (employees[msg.sender].employedSince == 0) revert EmployeeNotRegistered();
    _;
}
```
- **HR Manager Functions**: Protected by `onlyHrManager`
  - Employee registration
  - Employee termination
- **Employee Functions**: Protected by `onlyRegisteredEmployee`
  - Salary withdrawal
  - Currency preference switching
- **Zero-address Validation**
- **Employee Status Checks**

### Reentrancy Protection
```solidity
contract HumanResources is IHumanResources, ReentrancyGuard {
    function withdrawSalary() external nonReentrant {
        // Implementation
    }
}
```
- OpenZeppelin's ReentrancyGuard implementation
- State changes before external calls
- Secure withdrawal pattern
- Check-Effects-Interaction pattern

### Slippage Protection
```solidity
uint256 minOutputAmount = (estimatedWeiOutput * MIN_SLIPPAGE_PERCENT) / 100;
```
- 2% slippage tolerance
- Minimum output validation
- Price impact protection
- Short-term manipulation resistance

### Time-based Protection
```solidity
deadline: block.timestamp + SWAP_DEADLINE
```
- Short trade execution window (5 seconds)
- Accurate timestamp handling
- Employment period validation

## Error Handling

Custom error types for precise error reporting:
```solidity
error NotAuthorized()
error EmployeeAlreadyRegistered()
error EmployeeNotRegistered()
error NoSalaryAvailable()
error ExcessiveSlippage()
error EthTransferFailed()
error UsdcTransferFailed()
error UsdcApprovalFailed()
```

## Events

```solidity
event EmployeeRegistered(address indexed employee, uint256 weeklyUsdSalary)
event EmployeeTerminated(address indexed employee)
event SalaryWithdrawn(address indexed employee, bool isEth, uint256 amount)
event CurrencySwitched(address indexed employee, bool isEth)