// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.24;

enum PaymentType {Donate, Withdraw}


// враппер юзера, чтобы иметь быстрый доступ к его балансу
struct UserBank{
    User user;
    uint currentBalance;
}


struct User {
    string uuid;
    string name;
    Payment[] payments; //транзакции
    Wish[] wishes; 
    string[] topics; //темы стримера
 }


//транзакция
struct Payment {
    string uuid;
    PaymentUserData paymentUserData; 
    PaymentInfo paymentInfo;
    uint amount;
    uint transferedToUserAmount;
}

struct Wish {
    string userUUID;
    string wishUUID; // изменено с uint id на string wishUUID
    uint currentBalance; //текущее кол-во собранных денег
    uint price; 
    string name;
    string link; //ссылка на товар
    string description; 
    bool completed;
    // string image; //путь на картинку (хз зачем она тут, мб уберу ибо не используется)
}


// сообщения и имя, которые видит стример в транзакциях
struct PaymentUserData {
    string userName;
    string messageText;
}


// общая инфа о транзакциях
struct PaymentInfo{
    uint date;
    string fromUUID;
    string toUUID;
    uint wishId;
    address toAddress;
    PaymentType paymentType; // вывод/донат
}