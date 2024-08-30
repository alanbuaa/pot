syntax = "proto3";
package pb;

option go_package="./pb";

message SRS{

}

message PointG1{

}

message EncShare{

}

message SchnorrProof{

}

message DrawProof{

}

message PointG1List{
  repeated PointG1 PointG1s = 1;
}

message EncSharelist{
  repeated EncShare EncShares = 1;
}

message CryptoElement{
  SRS SRS = 1;
  repeated PointG1List PvssPkLists = 2;
  repeated PointG1List ShareCommitLists = 3;
  repeated PointG1List CoeffCommitLists = 4;
  repeated EncSharelist EncSharesLists = 5;

  SchnorrProof SrsUpdateProof = 6;
  DrawProof ShuffleProof = 7;
  DrawProof DrawProof = 8;
}
