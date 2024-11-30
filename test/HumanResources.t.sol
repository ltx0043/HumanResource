// SPDX-License-Identifier: GPL-3.0
pragma solidity ^0.8.24;

import {Test, console2} from "forge-std/Test.sol";
import {HumanResources} from "../src/HumanResources.sol";
import {IHumanResources} from "../src/interfaces/IHumanResources.sol";
import {IERC20} from "@openzeppelin/contracts/token/ERC20/IERC20.sol";

/**
 * @title HumanResources Contract Test Suite
 * @notice Comprehensive test suite for the HumanResources smart contract
 * @dev Tests include both specific scenarios and fuzz testing
 */
contract HumanResourcesTest is Test {
    /* ========== STATE VARIABLES ========== */

    HumanResources public humanResources;
    address public hrManager;
    address public employee1;
    address public employee2;
    
    /* ========== CONSTANTS ========== */

    // Optimism Mainnet Contract Addresses
    address constant USDC = 0x0b2C639c533813f4Aa9D7837CAf62653d097Ff85;
    address constant WETH = 0x4200000000000000000000000000000000000006;
    
    // Time Constants
    uint256 constant SECONDS_IN_WEEK = 7 days;
    uint256 constant DECIMAL_PLACES = 1e18;
    uint256 constant BASE_WEEKLY_SALARY = 1000 * DECIMAL_PLACES; // 1000 USD per week

    /* ========== SETUP ========== */

    function setUp() public {
        // Fork Optimism mainnet
        string memory rpc = vm.envString("RPC_URL");
        vm.createSelectFork(rpc);
        
        // Initialize test accounts
        hrManager = makeAddr("humanResources");
        employee1 = makeAddr("alice");
        employee2 = makeAddr("bob");
        
        // Deploy and fund contract
        humanResources = new HumanResources(hrManager);
        deal(USDC, address(humanResources), 1_000_000 * DECIMAL_PLACES);
        // Add ETH funding for ETH-based salary payments
        deal(address(humanResources), 1000 ether);
        // Fund WETH for swaps
        deal(WETH, address(humanResources), 1000 ether);
    }

    /* ========== CORE FUNCTIONALITY TESTS ========== */

    /// @notice Test employee registration process
    function testEmployeeRegistration() public {
        vm.startPrank(hrManager);
        humanResources.registerEmployee(employee1, BASE_WEEKLY_SALARY);
        
        (uint256 salary, uint256 since, uint256 terminated) = humanResources.getEmployeeInfo(employee1);
        assertEq(salary, BASE_WEEKLY_SALARY, "Incorrect salary stored");
        assertEq(since, block.timestamp, "Incorrect start timestamp");
        assertEq(terminated, 0, "Should not be terminated");
        assertEq(humanResources.getActiveEmployeeCount(), 1, "Active count should be 1");
        assertEq(humanResources.hrManager(), hrManager, "Incorrect HR manager");
        vm.stopPrank();
    }

    /// @notice Test employee termination process
    function testEmployeeTermination() public {
        vm.startPrank(hrManager);
        humanResources.registerEmployee(employee1, BASE_WEEKLY_SALARY);
        humanResources.terminateEmployee(employee1);
        vm.stopPrank();
        
        (,, uint256 terminated) = humanResources.getEmployeeInfo(employee1);
        assertEq(terminated, block.timestamp, "Incorrect termination timestamp");
        assertEq(humanResources.getActiveEmployeeCount(), 0, "Active count should be 0");
    }

    /// @notice Test salary withdrawal in USDC
    function testSalaryWithdrawalInUSDC() public {
        vm.prank(hrManager);
        humanResources.registerEmployee(employee1, BASE_WEEKLY_SALARY);
        
        // Advance time by 4 days
        skip(4 days);
        
        uint256 expectedSalary = (BASE_WEEKLY_SALARY * 4 days) / SECONDS_IN_WEEK;
        uint256 scaledExpectedSalary = expectedSalary / 1e12;
        
        vm.prank(employee1);
        humanResources.withdrawSalary();
        
        assertEq(IERC20(USDC).balanceOf(employee1), scaledExpectedSalary, "Incorrect USDC withdrawal amount");
    }

    /// @notice Test currency preference switching
    function testCurrencyPreferenceSwitch() public {
        vm.prank(hrManager);
        humanResources.registerEmployee(employee1, BASE_WEEKLY_SALARY);
        
        // First withdraw in USDC
        skip(4 days);
        vm.prank(employee1);
        humanResources.withdrawSalary();
        
        uint256 initialBalance = employee1.balance;
        
        // Then switch to ETH and accumulate new salary
        vm.startPrank(employee1);
        humanResources.switchCurrency();
        skip(4 days);
        humanResources.withdrawSalary();
        vm.stopPrank();
        
        assertTrue(employee1.balance > initialBalance, "ETH balance should increase after withdrawal");
    }

    /// @notice Test withdrawal after currency switch with ETH funding
    function testWithdrawalAfterCurrencySwitch() public {
        vm.prank(hrManager);
        humanResources.registerEmployee(employee1, BASE_WEEKLY_SALARY);
        
        // First withdraw in USDC
        skip(2 days);
        vm.prank(employee1);
        humanResources.withdrawSalary();
        
        // Switch to ETH and accumulate more salary
        vm.startPrank(employee1);
        humanResources.switchCurrency();
        skip(2 days);
        humanResources.withdrawSalary();
        vm.stopPrank();
        
        assertTrue(employee1.balance > 0, "Should have received ETH");
    }

    /// @notice Test zero salary registration
    function testZeroSalaryRegistration() public {
        vm.startPrank(hrManager);
        humanResources.registerEmployee(employee1, 0);
        
        (uint256 salary,,) = humanResources.getEmployeeInfo(employee1);
        assertEq(salary, 0, "Salary should be zero");
        vm.stopPrank();
    }

    /// @notice Test multiple currency switches
    function testMultipleCurrencySwitches() public {
        vm.prank(hrManager);
        humanResources.registerEmployee(employee1, BASE_WEEKLY_SALARY);
        
        // First withdraw in USDC
        skip(1 days);
        vm.prank(employee1);
        humanResources.withdrawSalary();
        
        vm.startPrank(employee1);
        
        // Switch to ETH
        humanResources.switchCurrency();
        skip(1 days);
        uint256 initialEthBalance = employee1.balance;
        humanResources.withdrawSalary();
        assertTrue(employee1.balance > initialEthBalance, "Should have received ETH");
        
        // Switch back to USDC
        humanResources.switchCurrency();
        skip(1 days);
        uint256 initialUsdcBalance = IERC20(USDC).balanceOf(employee1);
        humanResources.withdrawSalary();
        assertTrue(IERC20(USDC).balanceOf(employee1) > initialUsdcBalance, "Should have received USDC");
        
        vm.stopPrank();
    }

    /// @notice Test withdrawal with minimum time period
    function testMinimumPeriodWithdrawal() public {
        vm.prank(hrManager);
        humanResources.registerEmployee(employee1, BASE_WEEKLY_SALARY);
        
        // Try to withdraw with minimum time (1 hour)
        skip(1 hours);
        
        vm.prank(employee1);
        humanResources.withdrawSalary();
        
        uint256 expectedSalary = (BASE_WEEKLY_SALARY * 1 hours) / SECONDS_IN_WEEK;
        uint256 scaledExpectedSalary = expectedSalary / 1e12;
        assertEq(IERC20(USDC).balanceOf(employee1), scaledExpectedSalary);
    }

    /* ========== FUZZ TESTING ========== */

    /// @notice Fuzz test salary calculations with various time periods
    function testFuzzSalaryCalculation(uint256 timePeriod) public {
        // Bound time period between 1 hour and 4 weeks
        timePeriod = bound(timePeriod, 1 hours, 4 weeks);
        
        vm.prank(hrManager);
        humanResources.registerEmployee(employee1, BASE_WEEKLY_SALARY);
        
        skip(timePeriod);
        
        uint256 expectedSalary = (BASE_WEEKLY_SALARY * timePeriod) / SECONDS_IN_WEEK;
        assertEq(
            humanResources.salaryAvailable(employee1),
            expectedSalary,
            "Salary calculation incorrect for time period"
        );
    }

    /// @notice Fuzz test salary amounts within reasonable bounds
    function testFuzzSalaryAmounts(uint256 weeklySalary) public {
        // Bound salary between $100 and $10000 per week
        weeklySalary = bound(weeklySalary, 100 * DECIMAL_PLACES, 10000 * DECIMAL_PLACES);
        
        vm.prank(hrManager);
        humanResources.registerEmployee(employee1, weeklySalary);
        
        skip(1 weeks);
        
        vm.prank(employee1);
        humanResources.withdrawSalary();
        
        uint256 scaledSalary = weeklySalary / 1e12;
        assertEq(
            IERC20(USDC).balanceOf(employee1),
            scaledSalary,
            "Incorrect salary withdrawal amount"
        );
    }

    /// @notice Fuzz test multiple withdrawals with varying time periods
    function testFuzzMultipleWithdrawals(uint256 firstPeriod, uint256 secondPeriod) public {
        // Bound periods between 1 day and 2 weeks each
        firstPeriod = bound(firstPeriod, 1 days, 2 weeks);
        secondPeriod = bound(secondPeriod, 1 days, 2 weeks);
        
        vm.prank(hrManager);
        humanResources.registerEmployee(employee1, BASE_WEEKLY_SALARY);
        
        // First withdrawal
        skip(firstPeriod);
        uint256 firstExpectedSalary = (BASE_WEEKLY_SALARY * firstPeriod) / SECONDS_IN_WEEK;
        uint256 scaledFirstSalary = firstExpectedSalary / 1e12;
        
        vm.prank(employee1);
        humanResources.withdrawSalary();
        
        // Second withdrawal
        skip(secondPeriod);
        uint256 secondExpectedSalary = (BASE_WEEKLY_SALARY * secondPeriod) / SECONDS_IN_WEEK;
        uint256 scaledSecondSalary = secondExpectedSalary / 1e12;
        
        vm.prank(employee1);
        humanResources.withdrawSalary();
        
        assertEq(
            IERC20(USDC).balanceOf(employee1),
            scaledFirstSalary + scaledSecondSalary,
            "Incorrect total withdrawal amount"
        );
    }

    /* ========== REVERT TESTS ========== */

    /// @notice Test unauthorized access attempts
    function testRevertUnauthorizedAccess() public {
        // Attempt unauthorized registration
        vm.expectRevert(IHumanResources.NotAuthorized.selector);
        humanResources.registerEmployee(employee1, BASE_WEEKLY_SALARY);
        
        // Attempt unauthorized withdrawal
        vm.expectRevert(IHumanResources.EmployeeNotRegistered.selector);
        vm.prank(employee1);
        humanResources.withdrawSalary();
    }

    /// @notice Test invalid state transitions
    function testRevertInvalidStateTransitions() public {
        // Attempt to terminate non-existent employee
        vm.prank(hrManager);
        vm.expectRevert(IHumanResources.EmployeeNotRegistered.selector);
        humanResources.terminateEmployee(employee1);
        
        // Register and terminate employee
        vm.startPrank(hrManager);
        humanResources.registerEmployee(employee1, BASE_WEEKLY_SALARY);
        humanResources.terminateEmployee(employee1);
        
        // Attempt to terminate already terminated employee
        vm.expectRevert(IHumanResources.EmployeeNotRegistered.selector);
        humanResources.terminateEmployee(employee1);
        vm.stopPrank();
    }

    /// @notice Test terminated employee withdrawal behavior
    function testTerminatedEmployeeWithdrawal() public {
        // Register and terminate employee
        vm.startPrank(hrManager);
        humanResources.registerEmployee(employee1, BASE_WEEKLY_SALARY);
        skip(2 days);
        humanResources.terminateEmployee(employee1);
        vm.stopPrank();
        
        // Verify terminated employee can still withdraw accumulated salary
        uint256 expectedSalary = (BASE_WEEKLY_SALARY * 2 days) / SECONDS_IN_WEEK;
        uint256 scaledExpectedSalary = expectedSalary / 1e12;
        
        vm.prank(employee1);
        humanResources.withdrawSalary();
        
        assertEq(IERC20(USDC).balanceOf(employee1), scaledExpectedSalary);
    }
}