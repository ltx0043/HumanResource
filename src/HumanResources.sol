// SPDX-License-Identifier: GPL-3.0
pragma solidity ^0.8.24;

import { IHumanResources } from "./interfaces/IHumanResources.sol";
import { IWETH } from "./interfaces/IWETH.sol";
import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/utils/ReentrancyGuard.sol";
import "@chainlink/contracts/src/v0.8/shared/interfaces/AggregatorV3Interface.sol";
import "@uniswap/v3-periphery/contracts/interfaces/ISwapRouter.sol";
import "@uniswap/v3-periphery/contracts/interfaces/IQuoter.sol";


contract HumanResources is IHumanResources, ReentrancyGuard {
    /* ========== CONSTANTS ========== */

    uint24 public constant POOL_FEE_TIER = 3000;
    uint256 public constant MIN_SLIPPAGE_PERCENT = 98;
    uint256 public constant WEEK = 7 days;
    uint256 public constant PRICE_PRECISION = 1e12;
    uint256 public constant SWAP_DEADLINE = 5;
    uint256 private constant USDC_DECIMALS = 6;
    uint256 private constant STANDARD_DECIMALS = 18;
    uint256 private constant DECIMAL_CONVERTER = 10**(STANDARD_DECIMALS - USDC_DECIMALS); // 1e12

    /* ========== STATE VARIABLES ========== */

    uint256 public activeEmployeeCount;
    address public immutable hrManagerAddress;
    address public self = address(this);
    mapping(address => Employee) public employees;

    /* ========== EXTERNAL INTERFACES ========== */

    AggregatorV3Interface public constant CHAINKLINK_ETH_USDC = AggregatorV3Interface(0x13e3Ee699D1909E989722E753853AE30b17e08c5);
    IERC20 public constant USDC = IERC20(0x0b2C639c533813f4Aa9D7837CAf62653d097Ff85);
    IWETH public constant WETH = IWETH(0x4200000000000000000000000000000000000006);
    ISwapRouter public constant uniswapRouter = ISwapRouter(0xE592427A0AEce92De3Edee1F18E0157C05861564);

    /* ========== STRUCTS ========== */

    struct Employee {
        bool    prefersEth;
        uint256 weeklyUsdSalary;
        uint256 employedSince;
        uint256 terminatedAt;
        uint256 lastWithdrawal;
    }

    /* ========== ERRORS ========== */

    /** @notice Thrown when the slippage tolerance is exceeded during token swap */
    error ExcessiveSlippage();

    /** @notice Thrown when ETH transfer to employee fails */
    error EthTransferFailed();

    /** @notice Thrown when USDC transfer to employee fails */
    error UsdcTransferFailed();

    /** @notice Thrown when USDC approval for Uniswap fails */
    error UsdcApprovalFailed();

    /** @notice Thrown when employee tries to withdraw with zero salary available */
    error NoSalaryAvailable();

    /* ========== MODIFIERS ========== */

    modifier onlyHrManager() {
        if (msg.sender != hrManagerAddress) {
            revert NotAuthorized();
        }
        _;
    }

    modifier onlyRegisteredEmployee() {
        if (employees[msg.sender].employedSince == 0) {
            revert EmployeeNotRegistered();
        }
        _;
    }

    modifier validAddress(address account) {
        if (account == address(0)) {
            revert NotAuthorized();
        }
        _;
    }

    /* ========== CONSTRUCTOR ========== */

    constructor(address hr) {
        require(hr != address(0), "Invalid HR manager address");
        hrManagerAddress = hr;
    }

    /// @notice Register a new employee in the system with specified weekly salary
    /// @param employee Address of the employee to register
    /// @param weeklyUsdSalary Weekly salary amount in USD (18 decimals)
    function registerEmployee(address employee, uint256 weeklyUsdSalary) external onlyHrManager validAddress(employee) {
        if (employees[employee].employedSince != 0 && employees[employee].terminatedAt == 0) {
            revert EmployeeAlreadyRegistered();
        }

        employees[employee] = Employee(
            false,             // prefersEth
            weeklyUsdSalary,   // weeklyUsdSalary (stored in 18 decimals)
            block.timestamp,   // employedSince
            0,                 // terminatedAt
            block.timestamp    // lastWithdrawal
        );
        if (employees[employee].terminatedAt == 0) {
            ++activeEmployeeCount;
        }
        emit EmployeeRegistered(employee, weeklyUsdSalary);
    }

    /// @notice Terminate an employee's employment status
    /// @param employee Address of the employee to terminate
    function terminateEmployee(
        address employee
    ) external onlyHrManager validAddress(employee) {
        Employee storage emp = employees[employee];
        if (emp.employedSince == 0 || emp.terminatedAt != 0) {
            revert EmployeeNotRegistered();
        }

        --activeEmployeeCount;
        emp.terminatedAt = block.timestamp;
        emit EmployeeTerminated(employee);
    }

    /// @notice Allow employees to withdraw their available salary in their preferred currency
    function withdrawSalary() external nonReentrant onlyRegisteredEmployee {
        Employee storage emp = employees[msg.sender];
        uint256 amount = _calculateSalaryAvailable(msg.sender);
        
        if (amount == 0) {
            revert NoSalaryAvailable();
        }

        emp.lastWithdrawal = block.timestamp;

        if (emp.prefersEth) {
            swapUSDCToETH(_convertToUsdcDecimals(amount));
        } else {
            if (!USDC.transfer(msg.sender, _convertToUsdcDecimals(amount))) {
                revert UsdcTransferFailed();
            }
        }
        emit SalaryWithdrawn(msg.sender, emp.prefersEth, amount);
    }

    /// @notice Switch employee's salary payment preference between ETH and USDC
    function switchCurrency() external nonReentrant onlyRegisteredEmployee {
        Employee storage emp = employees[msg.sender];
        
        if (emp.terminatedAt != 0) {
            revert NotAuthorized();
        }

        uint256 availableSalary = _calculateSalaryAvailable(msg.sender);
        if (availableSalary > 0) {
            emp.lastWithdrawal = block.timestamp;
            
            if (emp.prefersEth) {
                swapUSDCToETH(_convertToUsdcDecimals(availableSalary));
            } else {
                if (!USDC.transfer(msg.sender, _convertToUsdcDecimals(availableSalary))) {
                    revert UsdcTransferFailed();
                }
            }
            emit SalaryWithdrawn(msg.sender, emp.prefersEth, availableSalary);
        }

        emp.prefersEth = !emp.prefersEth;
        emit CurrencySwitched(msg.sender, emp.prefersEth);
    }

    /// @notice Get the amount of salary available for withdrawal for a specific employee
    /// @param employee Address of the employee to check
    /// @return Available salary amount in USDC
    function salaryAvailable(address employee) external view returns (uint256) {
        return _calculateSalaryAvailable(employee);
    }

    /// @notice Get the current HR manager address
    function hrManager() external view override returns (address) {
        return hrManagerAddress;
    }

    /// @notice Get the current count of active employees
    /// @return Number of active employees
    function getActiveEmployeeCount() external view returns (uint256) {
        return activeEmployeeCount;
    }

    /// @notice Get employee's basic information
    /// @param employee Address of the employee
    /// @return weeklyUsdSalary Weekly salary in USDC
    /// @return employedSince Employment start timestamp
    /// @return terminatedAt Employment termination timestamp (0 if still employed)
    function getEmployeeInfo(
        address employee
    ) external view returns (
        uint256 weeklyUsdSalary,
        uint256 employedSince,
        uint256 terminatedAt
    ) {
        Employee memory emp = employees[employee];
        return (emp.weeklyUsdSalary, emp.employedSince, emp.terminatedAt);
    }

    /// @notice Calculate available salary for an employee based on time elapsed
    /// @param employee Address of the employee
    /// @return Available salary amount in USDC
    function _calculateSalaryAvailable(
        address employee
    ) internal view returns (uint256) {
        Employee storage emp = employees[employee];
        
        if (emp.employedSince == 0) {
            return 0;
        }

        uint256 endTime;
        if (emp.terminatedAt == 0) {
            endTime = block.timestamp;
        } else {
            endTime = emp.terminatedAt;
        }
        
        if (endTime <= emp.lastWithdrawal) {
            return 0;
        }

        uint256 timeElapsed = endTime - emp.lastWithdrawal;
        return (emp.weeklyUsdSalary * timeElapsed) / WEEK;
    }

    /// @notice Convert amount from 18 decimals to USDC's 6 decimals
    /// @param amount Amount with 18 decimals
    /// @return Amount converted to 6 decimals
    function _convertToUsdcDecimals(uint256 amount) internal pure returns (uint256) {
        return amount / DECIMAL_CONVERTER;
    }

    /// @notice Convert USDC to ETH using Uniswap and send to employee
    /// @param usdcAmount Amount of USDC to convert
    function swapUSDCToETH(uint256 usdcAmount) internal {
        (,int256 latestPrice,,,) = CHAINKLINK_ETH_USDC.latestRoundData();

        uint256 estimatedWeiOutput = (usdcAmount * PRICE_PRECISION) / (uint256(latestPrice));
        uint256 minOutputAmount = (estimatedWeiOutput * MIN_SLIPPAGE_PERCENT) / 100;
        if (!USDC.approve(address(uniswapRouter), usdcAmount)) {
            revert UsdcApprovalFailed();
        }
        
        ISwapRouter.ExactInputSingleParams memory params = ISwapRouter
            .ExactInputSingleParams({
                tokenIn: address(USDC),
                tokenOut: address(WETH),
                fee: POOL_FEE_TIER, 
                recipient: self,
                deadline: block.timestamp + SWAP_DEADLINE,
                amountIn: usdcAmount,
                amountOutMinimum: minOutputAmount,
                sqrtPriceLimitX96: 0
            });
        uint256 ethAmount = uniswapRouter.exactInputSingle(params);
        
        if (ethAmount < minOutputAmount) {
            revert ExcessiveSlippage();
        }

        WETH.withdraw(ethAmount);
        (bool success,) = msg.sender.call{value: ethAmount}("");
        if (!success) {
            revert EthTransferFailed();
        }
    }

    /// @notice Allow contract to receive ETH
    receive() external payable {}
}
