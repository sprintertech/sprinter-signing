// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity 0.8.20;

import "@openzeppelin/contracts/access/Ownable.sol";

/**
    @title Facilitates deposits and creation of deposit proposals, and deposit executions.
    @author ChainSafe Systems.
 */
contract Admin is Ownable {
    address public _MPCAddress;

    event StartKeygen();
    event EndKeygen();
    event KeyRefresh(string hash);

    error MPCAddressAlreadySet();
    error MPCAddressIsNotUpdatable();
    error MPCAddressZeroAddress();

    constructor() Ownable(msg.sender) {}

    /**
        @notice Once MPC address is set, this method can't be invoked anymore.
        It's used to trigger the belonging process on the MPC side which also handles keygen function calls order.
        @notice Only callable by owner
     */
    function startKeygen() external onlyOwner {
        if (_MPCAddress != address(0)) revert MPCAddressAlreadySet();
        emit StartKeygen();
    }

    /**
        @notice This method can be called only once, after the MPC address is set.
        @notice Only callable by owner.
        @param MPCAddress Address that will be set as MPC address.
     */
    function endKeygen(address MPCAddress) external onlyOwner {
        if(MPCAddress == address(0)) revert MPCAddressZeroAddress();
        if (_MPCAddress != address(0)) revert MPCAddressIsNotUpdatable();
        _MPCAddress = MPCAddress;
        emit EndKeygen();
    }

    /**
        @notice It's used to trigger the refresh process on the MPC side.
        @notice Only callable by owner
        @param hash Topology hash which prevents changes during refresh process.
     */
    function refreshKey(string memory hash) external onlyOwner {
        emit KeyRefresh(hash);
    }
}
