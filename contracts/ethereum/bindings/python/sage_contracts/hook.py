"""SageVerificationHook Python binding"""

from typing import Optional, Dict, Any, List
from web3 import Web3
from .base import ContractBase
import json
import os


class SageVerificationHook(ContractBase):
    """SageVerificationHook contract interface"""
    
    # Contract bytecode for deployment
    BYTECODE = "0x6080604052346100225761001161008c565b604051610b0461009a8239610b0490f35b600080fd5b61003b9061003e906001600160a01b031682565b90565b6001600160a01b031690565b61003b90610027565b61003b9061004a565b9061006c61003b61008892610053565b82546001600160a01b0319166001600160a01b03919091161790565b9055565b61009733600361005c565b56fe6080604052600436101561001257600080fd5b60003560e01c80633db059b6146100b257806344337ea1146100ad578063537df3b6146100a85780635a2a26bd146100a35780637b319ba11461009e5780637f6e8cbf146100995780638da5cb5b14610094578063b0ff63831461008f578063c847ad351461008a5763dbac26e9036100b75761044e565b610401565b6103e6565b6103ad565b610355565b610311565b6101f8565b610178565b61015b565b6100ef565b600080fd5b60009103126100b757565b6100d46100d46100d49290565b90565b6100d460056100c7565b6100d46100d7565b9052565b565b346100b7576100ff3660046100bc565b61011a61010a6100e1565b6040519182918290815260200190565b0390f35b6001600160a01b031690565b6101338161011e565b036100b757565b905035906100ed8261012a565b906020828203126100b7576100d49161013a565b346100b75761017361016e366004610147565b610908565b604051005b346100b75761017361018b366004610147565b61093a565b6100d49061011e906001600160a01b031682565b6100d490610190565b6100d4906101a4565b906101c0906101ad565b600052602052604060002090565b6100d4916008021c81565b906100d491546101ce565b60006101f36100d492826101b6565b6101d9565b346100b75761011a61010a61020e366004610147565b6101e4565b80610133565b905035906100ed82610213565b909182601f830112156100b75781359167ffffffffffffffff83116100b75760200192600183028401116100b757565b916060838303126100b75761026b8284610219565b92610279836020830161013a565b92604082013567ffffffffffffffff81116100b7576102989201610226565b9091565b60005b8381106102af5750506000910152565b818101518382015260200161029f565b6102e06102e96020936102f3936102d4815190565b80835293849260200190565b9586910161029c565b601f01601f191690565b0190565b90151581526040602082018190526100d4929101906102bf565b346100b75761032d610324366004610256565b929190916106fa565b9061011a61033a60405190565b928392836102f7565b6100d4906101f36001916000926101b6565b346100b75761011a61010a61036b366004610147565b610343565b6100d4916008021c61011e565b906100d49154610370565b6100d46000600361037d565b6100e99061011e565b6020810192916100ed9190610394565b346100b7576103bd3660046100bc565b61011a6103c8610388565b6040519182918261039d565b6100d4603c6100c7565b6100d46103d4565b346100b7576103f63660046100bc565b61011a61010a6103de565b346100b757610173610414366004610256565b9291909161080b565b6100d4916008021c5b60ff1690565b906100d4915461041d565b6100d4906104496002916000926101b6565b61042c565b346100b75761011a610469610464366004610147565b610437565b60405191829182901515815260200190565b6100d490610426565b6100d4905461047b565b634e487b7160e01b600052604160045260246000fd5b90601f01601f1916810190811067ffffffffffffffff8211176104c657604052565b61048e565b906100ed6104d860405190565b92836104a4565b67ffffffffffffffff81116104c657602090601f01601f19160190565b9061050e610509836104df565b6104cb565b918252565b61051d60136104fc565b721059191c995cdcc8189b1858dadb1a5cdd1959606a1b602082015290565b6100d4610513565b6100d49081565b6100d49054610544565b634e487b7160e01b600052601160045260246000fd5b9190820180921161057857565b610555565b610587601c6104fc565b7f526567697374726174696f6e20636f6f6c646f776e2061637469766500000000602082015290565b6100d461057d565b90600019905b9181191691161790565b906105d86100d46105df926100c7565b82546105b8565b9055565b6105ed60206104fc565b7f4461696c7920726567697374726174696f6e206c696d69742072656163686564602082015290565b6100d46105e3565b90826000939282370152565b9092919261063a610509826104df565b938185526020850190828401116100b7576100ed9261061e565b9080601f830112156100b7578160206100d49335910161062a565b9190916040818403126100b757803567ffffffffffffffff81116100b75783610699918301610654565b92602082013567ffffffffffffffff81116100b7576100d49201610654565b6106c260126104fc565b71125b9d985b1a590811125108199bdc9b585d60721b602082015290565b6100d46106b8565b6100d460006104fc565b6100d46106e8565b509061070f61070a8360026101b6565b610484565b6107ee57600192426107446107406100d461073261072d888a6101b6565b61054b565b61073a6103d4565b9061056b565b9190565b106107df5761075283610968565b6107bd575b61076661072d600094856101b6565b6107746107406100d46100d7565b10156107b05761078d826107939261079794019061066f565b506109e1565b1590565b6107a55750906100d46106f2565b9050906100d46106e0565b50509050906100d4610616565b6107da60006107d5856107cf836100c7565b926101b6565b6105c8565b610757565b505050506000906100d46105b0565b5050506000906100d461053c565b60001981146105785760010190565b5090506100ed91506108386108218260006101b6565b61083261082d8261054b565b6107fc565b906105c8565b6107d5429160016101b6565b6100d49061011e565b6100d49054610844565b6020808252600a908201526927b7363c9037bbb732b960b11b604082015260600190565b1561088257565b60405162461bcd60e51b81528061089b60048201610857565b0390fd5b6100ed906108c8336108c26108bc6108b7600361084d565b61011e565b9161011e565b1461087b565b6108f2565b9060ff906105be565b151590565b906108eb6100d46105df926108d6565b82546108cd565b6100ed9061090360019160026101b6565b6108db565b6100ed9061089f565b6100ed90610929336108c26108bc6108b7600361084d565b6100ed9061090360009160026101b6565b6100ed90610911565b634e487b7160e01b600052601260045260246000fd5b8115610963570490565b610943565b61099d6107406100d46201518061098961099761072d61098f4284866100c7565b90610959565b9760016101b6565b916100c7565b1190565b634e487b7160e01b600052603260045260246000fd5b906109c0825190565b8110156109ce570160200190565b6109a1565b516001600160f81b03191690565b80516109f0610740600a6100c7565b10610ac857600090610a12610a0d610a07846100c7565b836109b7565b6109d3565b610a29601960fa1b5b916001600160f81b03191690565b1415908115610aa1575b8115610a7a575b8115610a4c575b506100d45750600190565b610a659150610a0d90610a5f60036100c7565b906109b7565b610a72601d60f91b610a1b565b141538610a41565b9050610a8c610a0d610a0760026100c7565b610a99601960fa1b610a1b565b141590610a3a565b9050610ab3610a0d610a0760016100c7565b610ac0606960f81b610a1b565b141590610a33565b5060009056fea2646970667358221220147f2efefd4078813d5d0f102f8745ceef3313022bf6ebfbbf8f5fe6f9afef5d64736f6c63430008130033"
    
    def __init__(self, web3: Web3, address: str, private_key: Optional[str] = None):
        super().__init__(web3, address, private_key=private_key)
    
    def _load_default_abi(self) -> list:
        """Load the default ABI for SageVerificationHook"""
        abi_path = os.path.join(
            os.path.dirname(__file__),
            '..',
            '..',
            'abi',
            'SageVerificationHook.abi.json'
        )
        if os.path.exists(abi_path):
            with open(abi_path, 'r') as f:
                return json.load(f)
        return [{"inputs":[],"stateMutability":"nonpayable","type":"constructor"},{"inputs":[],"name":"MAX_REGISTRATIONS_PER_DAY","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"REGISTRATION_COOLDOWN","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"account","type":"address"}],"name":"addToBlacklist","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"bytes32","name":"","type":"bytes32"},{"internalType":"address","name":"agentOwner","type":"address"},{"internalType":"bytes","name":"","type":"bytes"}],"name":"afterRegister","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"bytes32","name":"","type":"bytes32"},{"internalType":"address","name":"agentOwner","type":"address"},{"internalType":"bytes","name":"data","type":"bytes"}],"name":"beforeRegister","outputs":[{"internalType":"bool","name":"success","type":"bool"},{"internalType":"string","name":"reason","type":"string"}],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"","type":"address"}],"name":"blacklisted","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"","type":"address"}],"name":"lastRegistrationTime","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"owner","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"","type":"address"}],"name":"registrationAttempts","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"account","type":"address"}],"name":"removeFromBlacklist","outputs":[],"stateMutability":"nonpayable","type":"function"}]
    
    @classmethod
    def deploy(
        cls,
        web3: Web3,
        private_key: str,
        *constructor_args,
        gas: Optional[int] = None,
        gas_price: Optional[int] = None
    ) -> 'SageVerificationHook':
        """Deploy a new instance of the contract"""
        account = web3.eth.account.from_key(private_key)
        
        # Create contract factory
        contract = web3.eth.contract(
            abi=cls(web3, '0x' + '0' * 40)._load_default_abi(),
            bytecode=cls.BYTECODE
        )
        
        # Build constructor transaction
        tx = contract.constructor(*constructor_args).build_transaction({
            'from': account.address,
            'nonce': web3.eth.get_transaction_count(account.address),
            'gas': gas or 5000000,
            'gasPrice': gas_price or web3.eth.gas_price
        })
        
        # Sign and send
        signed_tx = account.sign_transaction(tx)
        tx_hash = web3.eth.send_raw_transaction(signed_tx.rawTransaction)
        
        # Wait for receipt
        receipt = web3.eth.wait_for_transaction_receipt(tx_hash)
        
        return cls(web3, receipt.contractAddress, private_key=private_key)

    # Read-only functions
    
    def m_a_x__r_e_g_i_s_t_r_a_t_i_o_n_s__p_e_r__d_a_y(self):
        """MAX_REGISTRATIONS_PER_DAY - Read function"""
        return self.call_function('MAX_REGISTRATIONS_PER_DAY')
    
    def r_e_g_i_s_t_r_a_t_i_o_n__c_o_o_l_d_o_w_n(self):
        """REGISTRATION_COOLDOWN - Read function"""
        return self.call_function('REGISTRATION_COOLDOWN')
    
    def blacklisted(self, arg):
        """blacklisted - Read function"""
        return self.call_function('blacklisted', arg)
    
    def last_registration_time(self, arg):
        """lastRegistrationTime - Read function"""
        return self.call_function('lastRegistrationTime', arg)
    
    def owner(self):
        """owner - Read function"""
        return self.call_function('owner')
    
    def registration_attempts(self, arg):
        """registrationAttempts - Read function"""
        return self.call_function('registrationAttempts', arg)

    # State-changing functions
    
    def add_to_blacklist(self, account, gas: Optional[int] = None, gas_price: Optional[int] = None) -> Dict:
        """addToBlacklist - Write function (requires private key)"""
        return self.send_transaction('addToBlacklist', account, gas=gas, gas_price=gas_price)
    
    def after_register(self, arg, agentOwner, arg, gas: Optional[int] = None, gas_price: Optional[int] = None) -> Dict:
        """afterRegister - Write function (requires private key)"""
        return self.send_transaction('afterRegister', arg, agentOwner, arg, gas=gas, gas_price=gas_price)
    
    def before_register(self, arg, agentOwner, data, gas: Optional[int] = None, gas_price: Optional[int] = None) -> Dict:
        """beforeRegister - Write function (requires private key)"""
        return self.send_transaction('beforeRegister', arg, agentOwner, data, gas=gas, gas_price=gas_price)
    
    def remove_from_blacklist(self, account, gas: Optional[int] = None, gas_price: Optional[int] = None) -> Dict:
        """removeFromBlacklist - Write function (requires private key)"""
        return self.send_transaction('removeFromBlacklist', account, gas=gas, gas_price=gas_price)
