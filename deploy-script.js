const Web3 = require('web3');
const fs = require('fs');
const path = require('path');

const web3 = new Web3('http://geth-dev:8545');

async function deployContract() {
    try {
        // Получаем аккаунты
        const accounts = await web3.eth.getAccounts();
        console.log('Available accounts:', accounts);

        if (accounts.length === 0) {
            throw new Error('No accounts available');
        }

        const deployer = accounts[0];
        console.log('Deploying from account:', deployer);

        // Читаем скомпилированный контракт
        const contractName = 'DonationContract'; // Измените на имя вашего контракта
        const abiPath = `/output/${contractName}.abi`;
        const binPath = `/output/${contractName}.bin`;

        if (!fs.existsSync(abiPath) || !fs.existsSync(binPath)) {
            console.log('Contract files not found. Available files:');
            const files = fs.readdirSync('/output');
            console.log(files);
            return;
        }

        const abi = JSON.parse(fs.readFileSync(abiPath, 'utf8'));
        const bytecode = '0x' + fs.readFileSync(binPath, 'utf8');

        // Создаем контракт
        const contract = new web3.eth.Contract(abi);

        // Деплоим контракт
        console.log('Deploying contract...');
        const deployTx = contract.deploy({
            data: bytecode,
            arguments: [] // todo добавить аргументы конструктора если нужно
        });

        const gas = await deployTx.estimateGas({ from: deployer });
        console.log('Estimated gas:', gas);

        const deployedContract = await deployTx.send({
            from: deployer,
            gas: gas + 100000, // Добавляем запас
            gasPrice: '20000000000' // 20 gwei
        });

        console.log('Contract deployed at address:', deployedContract.options.address);

        // Сохраняем адрес контракта
        const deploymentInfo = {
            address: deployedContract.options.address,
            abi: abi,
            deployer: deployer,
            timestamp: new Date().toISOString()
        };

        fs.writeFileSync('/output/deployment.json', JSON.stringify(deploymentInfo, null, 2));
        console.log('Deployment info saved to deployment.json');

    } catch (error) {
        console.error('Deployment failed:', error);
        process.exit(1);
    }
}

deployContract();
