/**
 * ERC-8004 standalone registries: administrative functions must be owner-only.
 *
 * Background: ERC8004ValidationRegistry exposed addTrustedTeeKey,
 * removeTrustedTeeKey, setMinStake, setMinValidators, setConsensusThreshold
 * and setMaxValidatorsPerRequest without any access control, and
 * ERC8004ReputationRegistry.setValidationRegistry could be called by anyone
 * before the first assignment. Both contracts now inherit Ownable2Step.
 */

import { expect } from "chai";
import { parseEther, ZeroAddress } from "ethers";
import { network } from "hardhat";

const { ethers } = await network.connect();

describe("ERC8004 access control", function () {
  let owner, attacker, other;

  beforeEach(async function () {
    [owner, attacker, other] = await ethers.getSigners();
  });

  describe("ERC8004ValidationRegistry", function () {
    let registry;

    beforeEach(async function () {
      const Factory = await ethers.getContractFactory("ERC8004ValidationRegistry");
      registry = await Factory.deploy(parseEther("0.01"), 3, 66);
      await registry.waitForDeployment();
    });

    it("deployer becomes the owner", async function () {
      expect(await registry.owner()).to.equal(owner.address);
    });

    const cases = [
      ["addTrustedTeeKey", ["0x" + "11".repeat(32)]],
      ["removeTrustedTeeKey", ["0x" + "11".repeat(32)]],
      ["setMinStake", [parseEther("1")]],
      ["setMinValidators", [5]],
      ["setConsensusThreshold", [80]],
      ["setMaxValidatorsPerRequest", [50]],
    ];

    for (const [fn, args] of cases) {
      it(`${fn} reverts for a non-owner`, async function () {
        await expect(registry.connect(attacker)[fn](...args))
          .to.be.revertedWithCustomError(registry, "OwnableUnauthorizedAccount")
          .withArgs(attacker.address);
      });

      it(`${fn} succeeds for the owner`, async function () {
        await registry.connect(owner)[fn](...args);
      });
    }

    it("owner changes take effect", async function () {
      await registry.setMinStake(parseEther("2"));
      expect(await registry.minStake()).to.equal(parseEther("2"));
      await registry.setConsensusThreshold(90);
      expect(await registry.consensusThreshold()).to.equal(90n);
      const key = "0x" + "22".repeat(32);
      await registry.addTrustedTeeKey(key);
      expect(await registry.trustedTeeKeys(key)).to.equal(true);
      await registry.removeTrustedTeeKey(key);
      expect(await registry.trustedTeeKeys(key)).to.equal(false);
    });

    it("ownership transfer is two-step", async function () {
      await registry.transferOwnership(other.address);
      expect(await registry.owner()).to.equal(owner.address);
      await expect(registry.connect(other).setMinValidators(4))
        .to.be.revertedWithCustomError(registry, "OwnableUnauthorizedAccount");
      await registry.connect(other).acceptOwnership();
      expect(await registry.owner()).to.equal(other.address);
      await registry.connect(other).setMinValidators(4);
    });
  });

  describe("ERC8004ReputationRegistry", function () {
    let reputation;

    beforeEach(async function () {
      const Factory = await ethers.getContractFactory("ERC8004ReputationRegistry");
      reputation = await Factory.deploy(ZeroAddress);
      await reputation.waitForDeployment();
    });

    it("initial setValidationRegistry is owner-only", async function () {
      await expect(reputation.connect(attacker).setValidationRegistry(attacker.address))
        .to.be.revertedWithCustomError(reputation, "OwnableUnauthorizedAccount")
        .withArgs(attacker.address);
      await reputation.connect(owner).setValidationRegistry(other.address);
      expect(await reputation.validationRegistry()).to.equal(other.address);
    });

    it("after assignment only the owner or the current registry may change it", async function () {
      await reputation.setValidationRegistry(other.address);
      await expect(reputation.connect(attacker).setValidationRegistry(attacker.address))
        .to.be.revertedWithCustomError(reputation, "OwnableUnauthorizedAccount");
      // current validation registry may hand over
      await reputation.connect(other).setValidationRegistry(owner.address);
      expect(await reputation.validationRegistry()).to.equal(owner.address);
      // owner may still change it
      await reputation.connect(owner).setValidationRegistry(other.address);
      expect(await reputation.validationRegistry()).to.equal(other.address);
    });
  });
});
